// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	pathutils "github.com/onosproject/onos-config/pkg/utils/path"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"

	transactionstore "github.com/onosproject/onos-config/pkg/store/v2/transaction"
	"github.com/onosproject/onos-config/pkg/utils"
	valueutils "github.com/onosproject/onos-config/pkg/utils/v2/values"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	log.Infof("Received gNMI Set Request %+v", req)
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

	overrides, err := getTargetVersionOverrides(req)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	// If target version overrides is nil, let's create one so we can use it to push through
	// target type/version information from topo Configurable to downstream controllers
	// as part of the transaction.
	if overrides.Overrides == nil {
		overrides.Overrides = make(map[string]*configapi.TargetTypeVersion)
	}

	transactionStrategy, err := getTransactionStrategy(req)
	if err != nil {
		log.Warn(err)
		return nil, errors.Status(errors.NewInvalid(err.Error())).Err()
	}

	if len(req.GetUpdate())+len(req.GetReplace())+len(req.GetDelete()) < 1 {
		err = errors.NewInvalid("no updates, replace or deletes in SetRequest")
		log.Warn(err)
		return nil, errors.Status(err).Err()
	}

	// Delete
	for _, u := range req.GetDelete() {
		target, err := s.getTargetInfo(ctx, targets, overrides, u.GetTarget(), prefixTargetID)
		if err != nil {
			return nil, errors.Status(err).Err()
		}

		if err := s.doDelete(req.GetPrefix(), u, target); err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	// Replace
	for _, u := range req.GetReplace() {
		target, err := s.getTargetInfo(ctx, targets, overrides, u.Path.GetTarget(), prefixTargetID)
		if err != nil {
			return nil, errors.Status(err).Err()
		}

		if err := s.doUpdateOrReplace(ctx, req.GetPrefix(), u, target); err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	// Update - extract targets and their models
	for _, u := range req.GetUpdate() {
		target, err := s.getTargetInfo(ctx, targets, overrides, u.Path.GetTarget(), prefixTargetID)
		if err != nil {
			return nil, errors.Status(err).Err()
		}

		if err := s.doUpdateOrReplace(ctx, req.GetPrefix(), u, target); err != nil {
			log.Warn(err)
			return nil, errors.Status(err).Err()
		}
	}

	if s.gnmiSetSizeLimit > 0 {
		if len(targets) != 1 {
			return nil, errors.Status(errors.NewInvalid(
				"gNMI Set must contain only 1 target. Found: %d. GNMI_SET_SIZE_LIMIT=%d", len(targets), s.gnmiSetSizeLimit)).Err()
		}
		for target, chanages := range targets {
			if len(chanages.updates)+len(chanages.removes) > s.gnmiSetSizeLimit {
				return nil, errors.Status(errors.NewInvalid(
					"number of updates and deletes in a gNMI Set must not exceed %d. Target: %s Updates: %d, Deletes %d",
					s.gnmiSetSizeLimit, target, len(chanages.updates), len(chanages.removes))).Err()
			}
		}
	}

	transaction, err := newTransaction(targets, overrides, transactionStrategy, userName)
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

	for transactionEvent := range eventCh {
		if (transactionEvent.Transaction.TransactionStrategy.Synchronicity == configapi.TransactionStrategy_ASYNCHRONOUS &&
			transactionEvent.Transaction.Status.State == configapi.TransactionStatus_COMMITTED) ||
			(transactionEvent.Transaction.TransactionStrategy.Synchronicity == configapi.TransactionStrategy_SYNCHRONOUS &&
				transactionEvent.Transaction.Status.State == configapi.TransactionStatus_APPLIED) {
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
		} else if transactionEvent.Transaction.Status.State == configapi.TransactionStatus_FAILED {
			var err error
			if transactionEvent.Transaction.Status.Failure != nil {
				switch transactionEvent.Transaction.Status.Failure.Type {
				case configapi.Failure_UNKNOWN:
					err = errors.NewUnknown(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_CANCELED:
					err = errors.NewCanceled(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_NOT_FOUND:
					err = errors.NewNotFound(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_ALREADY_EXISTS:
					err = errors.NewAlreadyExists(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_UNAUTHORIZED:
					err = errors.NewUnauthorized(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_FORBIDDEN:
					err = errors.NewForbidden(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_CONFLICT:
					err = errors.NewConflict(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_INVALID:
					err = errors.NewInvalid(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_UNAVAILABLE:
					err = errors.NewUnavailable(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_NOT_SUPPORTED:
					err = errors.NewNotSupported(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_TIMEOUT:
					err = errors.NewTimeout(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_INTERNAL:
					err = errors.NewInternal(transactionEvent.Transaction.Status.Failure.Description)
				default:
					err = errors.NewUnknown(transactionEvent.Transaction.Status.Failure.Description)
				}
			} else {
				err = errors.NewUnknown("unknown failure occurred")
			}
			log.Errorf("Transaction failed", err)
			return nil, errors.Status(err).Err()
		}
	}
	return nil, ctx.Err()
}

func (s *Server) getTargetInfo(ctx context.Context, targets map[configapi.TargetID]*targetInfo,
	overrides *configapi.TargetVersionOverrides, idPrefix string, id configapi.TargetID) (*targetInfo, error) {
	targetID := configapi.TargetID(idPrefix)
	if len(id) > 0 {
		targetID = id
	}

	// Just return the target info if we already have it
	if target, found := targets[targetID]; found {
		return target, nil
	}

	var targetType configapi.TargetType
	var targetVersion configapi.TargetVersion

	// Fetch the Configurable aspect of the target topo entity
	configurable, err := s.getTargetConfigurable(ctx, topoapi.ID(targetID))
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	// Use the type/version overrides if they are specified
	if ttv, ok := overrides.Overrides[string(targetID)]; ok {
		targetType = ttv.TargetType
		targetVersion = ttv.TargetVersion
	} else {
		// Otherwise, extract the information from the Configurable aspect
		targetType = configapi.TargetType(configurable.Type)
		targetVersion = configapi.TargetVersion(configurable.Version)

		// Push through the information obtained from Configurable aspect via overrides,
		// so it's available for downstream processing
		overrides.Overrides[string(targetID)] = &configapi.TargetTypeVersion{TargetType: targetType, TargetVersion: targetVersion}
	}

	// Find the model plugin using the target type and version
	modelPlugin, ok := s.pluginRegistry.GetPlugin(targetType, targetVersion)
	if !ok {
		err := errors.NewNotFound("model %s (v%s) plugin not found", targetType, targetVersion)
		log.Warn(err)
		return nil, err
	}

	// Create and register target info using the model information
	target := &targetInfo{
		targetID:      targetID,
		targetVersion: configapi.TargetVersion(modelPlugin.GetInfo().Info.Version),
		targetType:    configapi.TargetType(modelPlugin.GetInfo().Info.Name),
		plugin:        modelPlugin,
		persistent:    configurable.Persistent,
		updates:       make(configapi.TypedValueMap),
		removes:       make([]string, 0),
	}
	targets[targetID] = target

	return target, nil
}

func (s *Server) getTargetConfigurable(ctx context.Context, targetID topoapi.ID) (topoapi.Configurable, error) {
	var configurable topoapi.Configurable
	target, err := s.topo.Get(ctx, targetID)
	if err != nil {
		log.Warn(err)
		return configurable, err
	}

	err = target.GetAspect(&configurable)
	if err != nil {
		log.Warn(err)
		return configurable, err
	}
	return configurable, nil
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
