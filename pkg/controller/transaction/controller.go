// Copyright 2021-present Open Networking Foundation.
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

package transaction

import (
	"context"
	"time"

	"github.com/onosproject/onos-config/pkg/utils/tree"

	"github.com/onosproject/onos-config/pkg/utils"

	"github.com/onosproject/onos-config/pkg/pluginregistry"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

const defaultTimeout = 30 * time.Second

var log = logging.GetLogger("controller", "transaction")

// NewController returns a new control relation  controller
func NewController(transactions transaction.Store, configurations configuration.Store, pluginRegistry *pluginregistry.PluginRegistry) *controller.Controller {
	c := controller.NewController("transaction")

	c.Watch(&Watcher{
		transactions: transactions,
	})

	c.Watch(&ConfigurationWatcher{
		transactions:   transactions,
		configurations: configurations,
	})

	c.Reconcile(&Reconciler{
		transactions:   transactions,
		configurations: configurations,
		pluginRegistry: pluginRegistry,
	})

	return c
}

// Reconciler reconciles transactions
type Reconciler struct {
	transactions   transaction.Store
	configurations configuration.Store
	pluginRegistry *pluginregistry.PluginRegistry
}

// Reconcile reconciles transactions
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	index := id.Value.(configapi.Index)
	transaction, err := r.transactions.GetByIndex(ctx, index)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile Transaction %d", index, err)
			return controller.Result{}, err
		}
		log.Debugf("Transaction %d not found", index)
		return controller.Result{}, nil
	}

	log.Infof("Reconciling Transaction %d", transaction.Index)
	log.Debug(transaction)

	// If the transaction revision has changed, set the transaction to the PENDING state
	if transaction.Revision > transaction.Status.Revision {
		log.Infof("Processing Transaction %d revision %d", transaction.Index, transaction.Revision)
		transaction.Status.State = configapi.TransactionState_TRANSACTION_PENDING
		transaction.Status.Revision = transaction.Revision
		log.Debug(transaction.Status)
		err := r.transactions.UpdateStatus(ctx, transaction)
		if err != nil {
			if !errors.IsNotFound(err) && !errors.IsConflict(err) {
				log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
				return controller.Result{}, err
			}
			log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	}

	// If the transaction is in a completed state, queue the next transaction to be reconciled
	if transaction.Status.State == configapi.TransactionState_TRANSACTION_COMPLETE ||
		transaction.Status.State == configapi.TransactionState_TRANSACTION_FAILED {
		log.Debugf("Queueing next Transaction %d for reconciliation", transaction.Index+1)
		return controller.Result{Requeue: controller.NewID(transaction.Index + 1)}, nil
	}

	if ok, err := r.reconcileTransaction(ctx, transaction); err != nil {
		return controller.Result{}, err
	} else if ok {
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcileTransaction(ctx context.Context, transaction *configapi.Transaction) (bool, error) {
	switch t := transaction.Transaction.(type) {
	case *configapi.Transaction_Change:
		return r.reconcileTransactionChange(ctx, transaction, t.Change)
	case *configapi.Transaction_Rollback:
		return r.reconcileTransactionRollback(ctx, transaction, t.Rollback)
	}
	return false, nil
}

func (r *Reconciler) reconcileTransactionChange(ctx context.Context, transaction *configapi.Transaction, change *configapi.TransactionChange) (bool, error) {
	switch transaction.Status.State {
	case configapi.TransactionState_TRANSACTION_PENDING:
		// If the transaction is Pending, begin validation if the prior transaction
		// has already been applied. This simplifies concurrency control in the controller
		// and guarantees transactions are applied to the configurations in sequential order.
		return r.reconcileTransactionPending(ctx, transaction)
		// if the transaction is in the Validating state, Validate the changes in a transaction and
		// change the state to Applying if validation passed otherwise fail the transaction
	case configapi.TransactionState_TRANSACTION_VALIDATING:
		return r.reconcileTransactionChangeValidating(ctx, transaction, change)
		// If the transaction is in the Applying state, update the Configuration for each
		// target and Complete the transaction.
	case configapi.TransactionState_TRANSACTION_APPLYING:
		return r.reconcileTransactionChangeApplying(ctx, transaction, change)
	}
	return false, nil
}

func (r *Reconciler) reconcileTransactionPending(ctx context.Context, transaction *configapi.Transaction) (bool, error) {
	log.Debugf("Checking preconditions for Transaction %d", transaction.Index)
	prevTransaction, err := r.transactions.GetByIndex(ctx, transaction.Index-1)
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	}
	if errors.IsNotFound(err) ||
		prevTransaction.Status.State == configapi.TransactionState_TRANSACTION_COMPLETE ||
		prevTransaction.Status.State == configapi.TransactionState_TRANSACTION_FAILED {
		transaction.Status.State = configapi.TransactionState_TRANSACTION_VALIDATING
		log.Infof("Preparing Transaction %d for validation", transaction.Index)
		log.Debug(transaction.Status)
		err = r.transactions.UpdateStatus(ctx, transaction)
		if err != nil {
			if !errors.IsNotFound(err) && !errors.IsConflict(err) {
				log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
				return false, err
			}
			log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

func (r *Reconciler) reconcileTransactionChangeValidating(ctx context.Context, transaction *configapi.Transaction, change *configapi.TransactionChange) (bool, error) {
	log.Infof("Validating change Transaction %d", transaction.Index)
	// Look through the change targets and validate changes for each target
	transaction.Status.Sources = make(map[configapi.TargetID]configapi.Source)
	for targetID, change := range change.Changes {
		modelName := utils.ToModelNameV2(change.TargetType, change.TargetVersion)
		modelPlugin, ok := r.pluginRegistry.GetPlugin(modelName)
		if !ok {
			return false, errors.NewNotFound("model plugin not found")
		}

		pathValues := make([]*configapi.PathValue, 0, len(change.Values))
		for path, changeValue := range change.Values {
			pathValue := &configapi.PathValue{
				Path:    path,
				Value:   changeValue.Value,
				Deleted: changeValue.Delete,
				Index:   transaction.Index,
			}
			pathValues = append(pathValues, pathValue)
		}

		jsonTree, err := tree.BuildTree(pathValues, true)
		if err != nil {
			return false, err
		}
		// If validation fails any target, mark the transaction Failed.
		// If validation is successful, proceed to Applying.
		err = modelPlugin.Validate(ctx, jsonTree)
		if err != nil {
			log.Warnf("Failed validating Transaction %d", transaction.Index, err)
			transaction.Status.State = configapi.TransactionState_TRANSACTION_FAILED
			log.Debug(transaction.Status)
			err = r.transactions.UpdateStatus(ctx, transaction)
			if err != nil {
				if !errors.IsNotFound(err) && !errors.IsConflict(err) {
					log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
					return false, err
				}
				log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
				return false, nil
			}
			return true, nil
		}

		// Get the target configuration and record the source values in the transaction status
		var configValues map[string]*configapi.PathValue
		configID := configuration.NewID(targetID, change.TargetType, change.TargetVersion)
		if config, err := r.configurations.Get(ctx, configID); err != nil {
			if !errors.IsNotFound(err) {
				return false, err
			}
			configValues = make(map[string]*configapi.PathValue)
		} else if config.Values != nil {
			configValues = config.Values
		} else {
			configValues = make(map[string]*configapi.PathValue)
		}

		source := configapi.Source{
			TargetType:    change.TargetType,
			TargetVersion: change.TargetVersion,
			Values:        make(map[string]configapi.PathValue),
		}
		for path := range change.Values {
			pathValue, ok := configValues[path]
			if ok {
				source.Values[path] = *pathValue
			}
		}
		transaction.Status.Sources[targetID] = source
	}

	// Store configuration sources and move the transaction to the APPLYING state
	log.Infof("Successfully validated change Transaction %d", transaction.Index)
	transaction.Status.State = configapi.TransactionState_TRANSACTION_APPLYING
	log.Debug(transaction.Status)
	err := r.transactions.UpdateStatus(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
			return false, err
		}
		log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
		return false, nil
	}
	return true, nil
}

func (r *Reconciler) reconcileTransactionChangeApplying(ctx context.Context, transaction *configapi.Transaction, change *configapi.TransactionChange) (bool, error) {
	log.Infof("Applying change Transaction %d", transaction.Index)
	// Once the source configurations have been stored we can update the target configurations
	for targetID, change := range change.Changes {
		log.Infof("Applying change Transaction %d to target '%s'", transaction.Index, targetID)
		configID := configuration.NewID(targetID, change.TargetType, change.TargetVersion)
		config, err := r.configurations.Get(ctx, configID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed applying Transaction %d to target '%s'", transaction.Index, targetID, err)
				return false, err
			}

			log.Infof("Initializing Configuration for target '%s'", targetID)
			config = &configapi.Configuration{
				ID:            configID,
				TargetID:      targetID,
				TargetType:    change.TargetType,
				TargetVersion: change.TargetVersion,
				Values:        make(map[string]*configapi.PathValue),
			}
			for path, changeValue := range change.Values {
				config.Values[path] = &configapi.PathValue{
					Path:    path,
					Value:   changeValue.Value,
					Deleted: changeValue.Delete,
					Index:   transaction.Index,
				}
			}

			log.Debug(config)
			err = r.configurations.Create(ctx, config)
			if err != nil {
				if !errors.IsAlreadyExists(err) {
					log.Errorf("Failed initializing Configuration for target '%s'", targetID, err)
					return false, err
				}
				return false, nil
			}
		} else {
			log.Infof("Updating Configuration for target '%s'", targetID)
			if config.Values == nil {
				config.Values = make(map[string]*configapi.PathValue)
			}
			for path, changeValue := range change.Values {
				config.Values[path] = &configapi.PathValue{
					Path:    path,
					Value:   changeValue.Value,
					Deleted: changeValue.Delete,
					Index:   transaction.Index,
				}
			}
			log.Debug(config)
			err = r.configurations.Update(ctx, config)
			if err != nil {
				if !errors.IsConflict(err) && !errors.IsNotFound(err) {
					log.Errorf("Failed updating Configuration for target '%s'", targetID, err)
					return false, err
				}
				return false, nil
			}
		}
	}

	// Complete the transaction once the target configurations have been updated
	log.Infof("Completed applying change Transaction %d", transaction.Index)
	transaction.Status.State = configapi.TransactionState_TRANSACTION_COMPLETE
	log.Debug(transaction.Status)
	err := r.transactions.UpdateStatus(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
			return false, err
		}
		log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
		return false, nil
	}
	return true, nil
}

func (r *Reconciler) reconcileTransactionRollback(ctx context.Context, transaction *configapi.Transaction, rollback *configapi.TransactionRollback) (bool, error) {
	log.Debugf("Reconciling transaction rollback %s in %s state", transaction.ID, transaction.Status.State)
	switch transaction.Status.State {
	case configapi.TransactionState_TRANSACTION_PENDING:
		// If the transaction is Pending, begin validation if the prior transaction
		// has already been applied. This simplifies concurrency control in the controller
		// and guarantees transactions are applied to the configurations in sequential order.
		return r.reconcileTransactionPending(ctx, transaction)
		// if the transaction is in the Validating state, Validate the rollback in a transaction and
		// change the state to Applying if validation passed otherwise fail the transaction
	case configapi.TransactionState_TRANSACTION_VALIDATING:
		return r.reconcileTransactionRollbackValidating(ctx, transaction, rollback)
		// If the transaction is in the Applying state, rollback the Configuration for each
		// target and Complete the transaction.
	case configapi.TransactionState_TRANSACTION_APPLYING:
		return r.reconcileTransactionRollbackApplying(ctx, transaction, rollback)
	}
	return false, nil
}

func (r *Reconciler) reconcileTransactionRollbackValidating(ctx context.Context, transaction *configapi.Transaction, rollback *configapi.TransactionRollback) (bool, error) {
	log.Infof("Validating rollback Transaction %d", transaction.Index)
	// Get the transaction being rolled back and apply its sources to this transaction
	// The source transaction's sources are stored in the rollback transaction to ensure
	// the rollback can be applied once it's in the APPLYING state even if the source
	// transaction is deleted from the log during compaction.
	targetTransaction, err := r.transactions.GetByIndex(ctx, rollback.Index)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed validating rollback Transaction %d", transaction.Index, err)
			return false, err
		}
		log.Warnf("Rollback Transaction %d failed: target Transaction %d not found", transaction.Index, rollback.Index)
		transaction.Status.State = configapi.TransactionState_TRANSACTION_FAILED
	} else {
		switch t := targetTransaction.Transaction.(type) {
		case *configapi.Transaction_Change:
			// Loop through the target transaction's sources and validate the rollback.
			// A rollback is only valid if the affected paths have not been changed
			// since the target transaction.
			for targetID, targetSource := range targetTransaction.Status.Sources {
				configID := configuration.NewID(targetID, targetSource.TargetType, targetSource.TargetVersion)
				config, err := r.configurations.Get(ctx, configID)
				if err != nil {
					if !errors.IsNotFound(err) {
						log.Errorf("Failed validating rollback Transaction %d for target '%s'", transaction.Index, targetID, err)
						return false, err
					}
					return false, nil
				}
				if config.Values == nil {
					config.Values = make(map[string]*configapi.PathValue)
				}

				for path := range targetSource.Values {
					configValue, ok := config.Values[path]
					if !ok || configValue.Index != rollback.Index {
						log.Warnf("Rollback Transaction %d failed: target Transaction %d is superseded by one or more later Transactions", transaction.Index, rollback.Index)
						transaction.Status.State = configapi.TransactionState_TRANSACTION_FAILED
						log.Debug(transaction.Status)
						err = r.transactions.UpdateStatus(ctx, transaction)
						if err != nil {
							if !errors.IsNotFound(err) && !errors.IsConflict(err) {
								log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
								return false, err
							}
							log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
							return false, nil
						}
						return true, nil
					}
				}
			}

			// Compute the rollback sources, which includes deletes for paths that were unset
			// prior to the target transaction being applied.
			transaction.Status.Sources = make(map[configapi.TargetID]configapi.Source)
			for targetID, targetChange := range t.Change.Changes {
				targetSource := targetTransaction.Status.Sources[targetID]
				source := configapi.Source{
					TargetType:    targetSource.TargetType,
					TargetVersion: targetSource.TargetVersion,
					Values:        make(map[string]configapi.PathValue),
				}
				for path := range targetChange.Values {
					pathValue, ok := targetSource.Values[path]
					if ok {
						source.Values[path] = pathValue
					} else {
						source.Values[path] = configapi.PathValue{
							Path:    path,
							Deleted: true,
						}
					}
				}
				transaction.Status.Sources[targetID] = source
			}
			log.Infof("Successfully validated rollback Transaction %d", transaction.Index)
			transaction.Status.State = configapi.TransactionState_TRANSACTION_APPLYING
		default:
			log.Warnf("Rollback Transaction %d failed: target Transaction %d is not a change", transaction.Index, rollback.Index)
			transaction.Status.State = configapi.TransactionState_TRANSACTION_FAILED
		}
	}

	log.Debug(transaction.Status)
	err = r.transactions.UpdateStatus(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
			return false, err
		}
		log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
		return false, nil
	}
	return true, nil
}

func (r *Reconciler) reconcileTransactionRollbackApplying(ctx context.Context, transaction *configapi.Transaction, rollback *configapi.TransactionRollback) (bool, error) {
	log.Infof("Applying rollback Transaction %d", transaction.Index)
	// Once the source configurations have been stored we can update the target configurations
	for targetID, source := range transaction.Status.Sources {
		log.Infof("Applying rollback Transaction %d to target '%s'", transaction.Index, targetID)
		configID := configuration.NewID(targetID, source.TargetType, source.TargetVersion)
		config, err := r.configurations.Get(ctx, configID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed applying rollback Transaction %d to target '%s'", transaction.Index, targetID, err)
				return false, err
			}
			return false, nil
		}

		log.Infof("Updating Configuration for target '%s'", targetID)

		// Update the configuration's values with the transaction index
		for path, pathValue := range source.Values {
			if config.Values == nil {
				config.Values = make(map[string]*configapi.PathValue)
			}
			config.Values[path] = &configapi.PathValue{
				Path:    path,
				Value:   pathValue.Value,
				Deleted: pathValue.Deleted,
				Index:   transaction.Index,
			}
		}

		log.Debug(config)
		err = r.configurations.Update(ctx, config)
		if err != nil {
			if !errors.IsConflict(err) && !errors.IsNotFound(err) {
				log.Errorf("Failed updating Configuration for target '%s'", targetID, err)
				return false, err
			}
			return false, nil
		}
	}

	// Complete the transaction once the target configurations have been updated
	log.Infof("Completed applying rollback Transaction %d", transaction.Index)
	transaction.Status.State = configapi.TransactionState_TRANSACTION_COMPLETE
	log.Debug(transaction.Status)
	err := r.transactions.UpdateStatus(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
			return false, err
		}
		log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
		return false, nil
	}
	return true, nil
}
