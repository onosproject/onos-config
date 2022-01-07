// Copyright 2022-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gnmi

import (
	"context"
	"fmt"
	"strings"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	pathutils "github.com/onosproject/onos-config/pkg/utils/path"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"

	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/utils"
	valueutils "github.com/onosproject/onos-config/pkg/utils/values/v2"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	log.Infof("Received gNMI Set Request %v", req)
	var userName string
	if md := metautils.ExtractIncoming(ctx); md != nil {
		log.Infof("gNMI Set() called by '%s (%s) (%s)'. Groups [%v]",
			md.Get("preferred_username"), md.Get("name"), md.Get("email"), md.Get("groups"))
		userName = md.Get("preferred_username")
		if userName == "" {
			userName = md.Get("name")
		}
		// TODO replace the following with fine grained RBAC using OpenPolicyAgent Regos
		if err := utils.TemporaryEvaluate(md); err != nil {
			log.Warn(err)
			return nil, errors.Status(errors.NewUnauthorized(err.Error())).Err()
		}
	}
	targetUpdates := make(mapTargetUpdates)
	targetRemoves := make(mapTargetRemoves)

	prefixTargetID := configapi.TargetID(req.GetPrefix().GetTarget())
	targetInfo := targetInfo{}

	extensions, err := extractExtensions(req)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}
	targetInfo.targetType = extensions.targetType
	targetInfo.targetVersion = extensions.targetVersion

	if len(req.GetUpdate())+len(req.GetReplace())+len(req.GetDelete()) < 1 {
		err = errors.NewInvalid("no updates, replace or deletes in SetRequest")
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	// Update - extract targets and their models
	for _, u := range req.GetUpdate() {
		targetID := configapi.TargetID(u.Path.GetTarget())
		if targetID == "" { //Try the prefix
			targetInfo.targetID = prefixTargetID
		} else {
			targetInfo.targetID = targetID
		}

		modelPlugin, err := s.getModelPlugin(ctx, topoapi.ID(targetInfo.targetID))
		if err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}

		targetUpdates[targetInfo.targetID], err = s.doUpdateOrReplace(ctx, req.GetPrefix(), u, targetUpdates, modelPlugin)
		if err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	// Replace
	for _, u := range req.GetReplace() {
		targetID := configapi.TargetID(u.Path.GetTarget())
		if targetID == "" { //Try the prefix
			targetInfo.targetID = prefixTargetID
		} else {
			targetInfo.targetID = targetID
		}

		modelPlugin, err := s.getModelPlugin(ctx, topoapi.ID(targetInfo.targetID))
		if err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}

		targetUpdates[targetInfo.targetID], err = s.doUpdateOrReplace(ctx, req.GetPrefix(), u, targetUpdates, modelPlugin)
		if err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	//Delete
	for _, u := range req.GetDelete() {
		targetID := configapi.TargetID(u.GetTarget())
		if targetID == "" { //Try the prefix
			targetInfo.targetID = prefixTargetID
		} else {
			targetInfo.targetID = targetID
		}
		modelPlugin, err := s.getModelPlugin(ctx, topoapi.ID(targetInfo.targetID))
		if err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}

		targetRemoves[targetInfo.targetID], err = s.doDelete(req.GetPrefix(), u, targetRemoves, modelPlugin)
		if err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	transaction, err := newTransaction(targetInfo, extensions, targetUpdates, targetRemoves, userName)
	if err != nil {
		return nil, errors.Status(err).Err()
	}

	err = s.transactions.Create(ctx, transaction)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	// Build the responses
	updateResults := make([]*gnmi.UpdateResult, 0)
	for _, change := range transaction.GetChange().Changes {
		targetID := change.TargetID
		for _, valueUpdate := range change.Values {
			var updateResult *gnmi.UpdateResult
			if valueUpdate.Delete {
				updateResult, err = newUpdateResult(valueUpdate.Path,
					string(targetID), gnmi.UpdateResult_DELETE)
				if err != nil {
					log.Warn(err)
					return nil, err
				}
			} else {
				updateResult, err = newUpdateResult(valueUpdate.Path,
					string(targetID), gnmi.UpdateResult_UPDATE)
				if err != nil {
					log.Warn(err)
					return nil, err
				}
			}
			updateResults = append(updateResults, updateResult)
		}
	}

	setResponse := &gnmi.SetResponse{
		Response:  updateResults,
		Timestamp: time.Now().Unix(),
		Extension: []*gnmi_ext.Extension{
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  ExtensionTransactionID,
						Msg: []byte(transaction.ID),
					},
				},
			},
		},
	}

	return setResponse, nil
}

func (s *Server) getModelPlugin(ctx context.Context, targetID topoapi.ID) (*pluginregistry.ModelPlugin, error) {
	target, err := s.topo.Get(ctx, targetID)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	targetConfigurableAspect := &topoapi.Configurable{}
	err = target.GetAspect(targetConfigurableAspect)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	modelName := utils.ToModelNameV2(configapi.TargetType(targetConfigurableAspect.Type), configapi.TargetVersion(targetConfigurableAspect.Version))
	modelPlugin, ok := s.pluginRegistry.GetPlugin(modelName)
	if !ok {
		err = errors.NewNotFound("model %s plugin not found", modelName)
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}
	return modelPlugin, nil
}

// This deals with either a path and a value (simple case) or a path with
// a JSON body which implies multiple paths and values.
func (s *Server) doUpdateOrReplace(ctx context.Context, prefix *gnmi.Path, u *gnmi.Update,
	targetUpdates mapTargetUpdates, modelPlugin *pluginregistry.ModelPlugin) (configapi.TypedValueMap, error) {
	targetID := configapi.TargetID(u.Path.GetTarget())
	if targetID == "" {
		targetID = configapi.TargetID(prefix.GetTarget())
	}
	prefixPath := utils.StrPath(prefix)
	path := utils.StrPath(u.Path)
	if prefixPath != "/" {
		path = fmt.Sprintf("%s%s", prefixPath, path)
	}

	updates, ok := targetUpdates[targetID]
	if !ok {
		updates = make(configapi.TypedValueMap)
	}

	jsonVal := u.GetVal().GetJsonVal()
	if jsonVal != nil {
		log.Debugf("Processing Json Value in set from base %s: %s", path, string(jsonVal))
		pathValues, err := modelPlugin.GetPathValues(ctx, prefixPath, jsonVal)
		if err != nil {
			return nil, err
		}

		if len(pathValues) == 0 {
			log.Warnf("no pathValues found for %s in %v", path, string(jsonVal))
		}

		for _, cv := range pathValues {
			updates[cv.Path] = &cv.Value
		}
	} else {
		_, rwPathElem, err := pathutils.FindPathFromModel(path, modelPlugin.ReadWritePaths, true)
		if err != nil {
			return nil, err
		}
		updateValue, err := valueutils.GnmiTypedValueToNativeType(u.Val, rwPathElem)
		if err != nil {
			return nil, err
		}
		if err = pathutils.CheckKeyValue(path, rwPathElem, updateValue); err != nil {
			return nil, err
		}
		updates[path] = updateValue
	}

	return updates, nil

}

func (s *Server) doDelete(prefix *gnmi.Path, u *gnmi.Path,
	targetRemoves mapTargetRemoves, modelPlugin *pluginregistry.ModelPlugin) ([]string, error) {
	target := configapi.TargetID(u.GetTarget())
	if target == "" {
		target = configapi.TargetID(prefix.GetTarget())
	}
	deletes, ok := targetRemoves[target]
	if !ok {
		deletes = make([]string, 0)
	}
	prefixPath := utils.StrPath(prefix)
	path := utils.StrPath(u)
	if prefixPath != "/" {
		path = fmt.Sprintf("%s%s", prefixPath, path)
	}
	// Checks for read only paths
	isExactMatch, rwPath, err := pathutils.FindPathFromModel(path, modelPlugin.ReadWritePaths, false)
	if err != nil {
		return nil, err
	}
	if isExactMatch && rwPath.IsAKey && !strings.HasSuffix(path, "]") { // In case an index attribute is given - take it off
		path = path[:strings.LastIndex(path, "/")]
	}
	deletes = append(deletes, path)
	return deletes, nil
}
