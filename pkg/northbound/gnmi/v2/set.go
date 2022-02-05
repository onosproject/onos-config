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
	"github.com/gogo/protobuf/proto"
	"strings"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	pathutils "github.com/onosproject/onos-config/pkg/utils/path"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"

	"github.com/onosproject/onos-config/pkg/pluginregistry"
	transactionstore "github.com/onosproject/onos-config/pkg/store/transaction"
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

	prefixTargetID := configapi.TargetID(req.GetPrefix().GetTarget())
	targets := make(map[configapi.TargetID]*targetInfo)

	transactionMode, err := getSetExtensions(req)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	if len(req.GetUpdate())+len(req.GetReplace())+len(req.GetDelete()) < 1 {
		err = errors.NewInvalid("no updates, replace or deletes in SetRequest")
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	// Update - extract targets and their models
	for _, u := range req.GetUpdate() {
		target, err := s.getTargetInfo(ctx, targets, u.Path.GetTarget(), prefixTargetID)
		if err != nil {
			return nil, errors.Status(err).Err()
		}

		if err := s.doUpdateOrReplace(ctx, req.GetPrefix(), u, target); err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	// Replace
	for _, u := range req.GetReplace() {
		target, err := s.getTargetInfo(ctx, targets, u.Path.GetTarget(), prefixTargetID)
		if err != nil {
			return nil, errors.Status(err).Err()
		}

		if err := s.doUpdateOrReplace(ctx, req.GetPrefix(), u, target); err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	//Delete
	for _, u := range req.GetDelete() {
		target, err := s.getTargetInfo(ctx, targets, u.GetTarget(), prefixTargetID)
		if err != nil {
			return nil, errors.Status(err).Err()
		}

		if err := s.doDelete(req.GetPrefix(), u, target); err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	transaction, err := newTransaction(targets, transactionMode, userName)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	err = s.transactions.Create(ctx, transaction)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}
	eventCh := make(chan configapi.TransactionEvent)
	err = s.transactions.Watch(ctx, eventCh, transactionstore.WithReplay(), transactionstore.WithTransactionID(transaction.ID))
	if err != nil {
		return nil, errors.Status(err).Err()
	}

	isSync := transactionMode != nil && transactionMode.Sync
	for transactionEvent := range eventCh {
		if transactionEvent.Transaction.Status.Phases.Initialize != nil &&
			transactionEvent.Transaction.Status.Phases.Initialize.State == configapi.TransactionInitializePhase_FAILED {
			err := getErrorFromFailure(transactionEvent.Transaction.Status.Phases.Initialize.Failure)
			log.Errorf("Transaction failed", err)
			return nil, errors.Status(err).Err()
		}

		if transactionEvent.Transaction.Status.Phases.Validate != nil &&
			transactionEvent.Transaction.Status.Phases.Validate.State == configapi.TransactionValidatePhase_FAILED {
			err := getErrorFromFailure(transactionEvent.Transaction.Status.Phases.Validate.Failure)
			log.Errorf("Transaction failed validation", err)
			return nil, errors.Status(err).Err()
		}

		if (!isSync && transactionEvent.Transaction.Status.Phases.Commit != nil &&
			transactionEvent.Transaction.Status.Phases.Commit.State == configapi.TransactionCommitPhase_COMMITTED) ||
			(isSync && transactionEvent.Transaction.Status.Phases.Apply != nil &&
				transactionEvent.Transaction.Status.Phases.Apply.State == configapi.TransactionApplyPhase_APPLIED) {
			updateResults := make([]*gnmi.UpdateResult, 0)
			for targetID, change := range transaction.GetChange().Values {
				for path, valueUpdate := range change.Values {
					var updateResult *gnmi.UpdateResult
					if valueUpdate.Deleted {
						updateResult, err = newUpdateResult(path, string(targetID), gnmi.UpdateResult_DELETE)
						if err != nil {
							log.Warn(err)
							return nil, err
						}
					} else {
						updateResult, err = newUpdateResult(path, string(targetID), gnmi.UpdateResult_UPDATE)
						if err != nil {
							log.Warn(err)
							return nil, err
						}
					}
					updateResults = append(updateResults, updateResult)
				}
			}

			transactionInfo := &configapi.TransactionInfo{
				ID:    transaction.ID,
				Index: transaction.Index,
			}
			transactionInfoBytes, err := proto.Marshal(transactionInfo)
			if err != nil {
				log.Warn(err)
				return nil, errors.Status(errors.NewInternal(err.Error())).Err()
			}

			response := &gnmi.SetResponse{
				Response:  updateResults,
				Timestamp: time.Now().Unix(),
				Extension: []*gnmi_ext.Extension{
					{
						Ext: &gnmi_ext.Extension_RegisteredExt{
							RegisteredExt: &gnmi_ext.RegisteredExtension{
								Id:  configapi.TransactionInfoExtensionID,
								Msg: transactionInfoBytes,
							},
						},
					},
				},
			}
			log.Debugf("Sending SetResponse %+v", response)
			return response, nil
		}
	}
	return nil, ctx.Err()
}

func getErrorFromFailure(failure *configapi.Failure) error {
	if failure == nil {
		return errors.NewUnknown("unknown failure occurred")
	}

	switch failure.Type {
	case configapi.Failure_UNKNOWN:
		return errors.NewUnknown(failure.Description)
	case configapi.Failure_CANCELED:
		return errors.NewCanceled(failure.Description)
	case configapi.Failure_NOT_FOUND:
		return errors.NewNotFound(failure.Description)
	case configapi.Failure_ALREADY_EXISTS:
		return errors.NewAlreadyExists(failure.Description)
	case configapi.Failure_UNAUTHORIZED:
		return errors.NewUnauthorized(failure.Description)
	case configapi.Failure_FORBIDDEN:
		return errors.NewForbidden(failure.Description)
	case configapi.Failure_CONFLICT:
		return errors.NewConflict(failure.Description)
	case configapi.Failure_INVALID:
		return errors.NewInvalid(failure.Description)
	case configapi.Failure_UNAVAILABLE:
		return errors.NewUnavailable(failure.Description)
	case configapi.Failure_NOT_SUPPORTED:
		return errors.NewNotSupported(failure.Description)
	case configapi.Failure_TIMEOUT:
		return errors.NewTimeout(failure.Description)
	case configapi.Failure_INTERNAL:
		return errors.NewInternal(failure.Description)
	default:
		return errors.NewUnknown(failure.Description)
	}
}

func (s *Server) getTargetInfo(ctx context.Context, targets map[configapi.TargetID]*targetInfo, idPrefix string, id configapi.TargetID) (*targetInfo, error) {
	targetID := configapi.TargetID(idPrefix)
	if len(id) > 0 {
		targetID = id
	}

	if target, found := targets[targetID]; found {
		return target, nil
	}

	modelPlugin, err := s.getModelPlugin(ctx, topoapi.ID(targetID))
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	target := &targetInfo{
		targetID:      targetID,
		targetVersion: configapi.TargetVersion(modelPlugin.GetInfo().Info.Version),
		targetType:    configapi.TargetType(modelPlugin.GetInfo().Info.Name),
		plugin:        modelPlugin,
		updates:       make(configapi.TypedValueMap),
		removes:       make([]string, 0),
	}
	targets[targetID] = target

	return target, nil
}

func (s *Server) getModelPlugin(ctx context.Context, targetID topoapi.ID) (pluginregistry.ModelPlugin, error) {
	target, err := s.topo.Get(ctx, targetID)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	targetConfigurableAspect := &topoapi.Configurable{}
	err = target.GetAspect(targetConfigurableAspect)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	modelPlugin, ok := s.pluginRegistry.GetPlugin(configapi.TargetType(targetConfigurableAspect.Type), configapi.TargetVersion(targetConfigurableAspect.Version))
	if !ok {
		err = errors.NewNotFound("model %s (v%s) plugin not found", targetConfigurableAspect.Type, targetConfigurableAspect.Version)
		log.Warn(err)
		return nil, err
	}
	return modelPlugin, nil
}

// This deals with either a path and a value (simple case) or a path with
// a JSON body which implies multiple paths and values.
func (s *Server) doUpdateOrReplace(ctx context.Context, prefix *gnmi.Path, u *gnmi.Update, target *targetInfo) error {
	prefixPath := utils.StrPath(prefix)
	path := utils.StrPath(u.Path)
	if prefixPath != "/" {
		path = fmt.Sprintf("%s%s", prefixPath, path)
	}

	jsonVal := u.GetVal().GetJsonVal()
	if jsonVal != nil {
		log.Debugf("Processing Json Value in set from base %s: %s", path, string(jsonVal))
		pathValues, err := target.plugin.GetPathValues(ctx, prefixPath, jsonVal)
		if err != nil {
			return err
		}

		if len(pathValues) == 0 {
			log.Warnf("no pathValues found for %s in %v", path, string(jsonVal))
		}

		for _, cv := range pathValues {
			target.updates[cv.Path] = &cv.Value
		}
	} else {
		_, rwPathElem, err := pathutils.FindPathFromModel(path, target.plugin.GetInfo().ReadWritePaths, true)
		if err != nil {
			return err
		}
		updateValue, err := valueutils.GnmiTypedValueToNativeType(u.Val, rwPathElem)
		if err != nil {
			return err
		}
		if err = pathutils.CheckKeyValue(path, rwPathElem, updateValue); err != nil {
			return err
		}
		target.updates[path] = updateValue
	}

	return nil
}

func (s *Server) doDelete(prefix *gnmi.Path, gnmiPath *gnmi.Path, target *targetInfo) error {
	prefixPath := utils.StrPath(prefix)
	path := utils.StrPath(gnmiPath)
	if prefixPath != "/" {
		path = fmt.Sprintf("%s%s", prefixPath, path)
	}
	// Checks for read only paths
	isExactMatch, rwPath, err := pathutils.FindPathFromModel(path, target.plugin.GetInfo().ReadWritePaths, false)
	if err != nil {
		return err
	}
	if isExactMatch && rwPath.IsAKey && !strings.HasSuffix(path, "]") { // In case an index attribute is given - take it off
		path = path[:strings.LastIndex(path, "/")]
	}
	target.removes = append(target.removes, path)
	return nil
}
