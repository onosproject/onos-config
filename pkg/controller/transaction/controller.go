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
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

const defaultTimeout = 30 * time.Second

var log = logging.GetLogger("controller", "transaction")

// NewController returns a new control relation  controller
func NewController(topo topo.Store, transactions transaction.Store, configurations configuration.Store, pluginRegistry *pluginregistry.PluginRegistry) *controller.Controller {
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
		topo:           topo,
		configurations: configurations,
		pluginRegistry: pluginRegistry,
	})

	return c
}

// Reconciler reconciles transactions
type Reconciler struct {
	topo           topo.Store
	transactions   transaction.Store
	configurations configuration.Store
	pluginRegistry *pluginregistry.PluginRegistry
}

// Reconcile reconciles transactions
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	transactionID := id.Value.(configapi.TransactionID)
	transaction, err := r.transactions.Get(ctx, transactionID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile transaction %s, %s", transactionID, err)
			return controller.Result{}, err
		}
		log.Debugf("Transaction %s not found", transactionID)
		return controller.Result{}, nil
	}

	log.Infof("Reconciling transaction %v", transaction)

	// If the transaction is in a completed state, queue the next transaction to be reconciled
	if transaction.Status.State == configapi.TransactionState_TRANSACTION_COMPLETE ||
		transaction.Status.State == configapi.TransactionState_TRANSACTION_FAILED {
		nextTransaction, err := r.transactions.GetByIndex(ctx, transaction.Index+1)
		if err != nil {
			if !errors.IsNotFound(err) {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		log.Debugf("Queueing next transaction %d", nextTransaction.Index)
		return controller.Result{Requeue: controller.ID{Value: nextTransaction.ID}}, nil
	}

	if ok, err := r.reconcileTransaction(ctx, transaction); err != nil {
		log.Warnf("Failed to reconcile transaction: %s, %s", transactionID, err)
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
	log.Debugf("Reconciling transaction change %s in %s state", transaction.ID, transaction.Status.State)
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
	prevTransaction, err := r.transactions.GetByIndex(ctx, transaction.Index-1)
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	}
	if errors.IsNotFound(err) ||
		prevTransaction.Status.State == configapi.TransactionState_TRANSACTION_COMPLETE ||
		prevTransaction.Status.State == configapi.TransactionState_TRANSACTION_FAILED {
		transaction.Status.State = configapi.TransactionState_TRANSACTION_VALIDATING
		err = r.transactions.Update(ctx, transaction)
		if err != nil {
			if !errors.IsConflict(err) || !errors.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

func (r *Reconciler) reconcileTransactionChangeValidating(ctx context.Context, transaction *configapi.Transaction, change *configapi.TransactionChange) (bool, error) {
	for _, change := range change.Changes {
		modelName := utils.ToModelNameV2(change.TargetType, change.TargetVersion)
		modelPlugin, ok := r.pluginRegistry.GetPlugin(modelName)
		if !ok {
			return false, errors.NewNotFound("model plugin not found")
		}

		pathValues := make([]*configapi.PathValue, len(change.Values))
		for i, changeValue := range change.Values {
			pathValue := &configapi.PathValue{
				Path:    changeValue.Path,
				Value:   changeValue.Value,
				Deleted: changeValue.Delete,
			}
			pathValues[i] = pathValue
		}

		jsonTree, err := tree.BuildTree(pathValues, true)
		if err != nil {
			return false, err
		}
		// If validation fails any target, mark the transaction Failed.
		// If validation is successful, proceed to Applying.
		err = modelPlugin.Validate(ctx, jsonTree)
		if err != nil {
			transaction.Status.State = configapi.TransactionState_TRANSACTION_FAILED
			err = r.transactions.Update(ctx, transaction)
			if err != nil {
				if !errors.IsConflict(err) && !errors.IsNotFound(err) {
					return false, err
				}
				return false, nil
			}
			return true, nil
		}
	}

	transaction.Status.State = configapi.TransactionState_TRANSACTION_APPLYING
	err := r.transactions.Update(ctx, transaction)
	if err != nil {
		if !errors.IsConflict(err) && !errors.IsNotFound(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (r *Reconciler) reconcileTransactionChangeApplying(ctx context.Context, transaction *configapi.Transaction, change *configapi.TransactionChange) (bool, error) {
	transaction.Status.Sources = make(map[configapi.TargetID]*configapi.Source)
	configs := make(map[configapi.TargetID]*configapi.Configuration)
	for _, change := range change.Changes {
		config, err := r.configurations.Get(ctx, change.TargetID)
		if err != nil {
			if !errors.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}
		configs[config.TargetID] = config
		source := &configapi.Source{
			Values: make(map[string]configapi.Index),
		}
		for _, changeValue := range change.Values {
			pathValue, ok := config.Values[changeValue.Path]
			if ok {
				source.Values[pathValue.Path] = pathValue.Index
			}
		}
		transaction.Status.Sources[config.TargetID] = source
	}

	// The source configurations must be stored prior to updating the target configurations
	// otherwise they will be lost.
	err := r.transactions.Update(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			return false, err
		}
		return false, nil
	}

	// Once the source configurations have been stored we can update the target configurations
	for _, change := range change.Changes {
		config := configs[change.TargetID]
		for _, changeValue := range change.Values {
			if config.Values == nil {
				config.Values = make(map[string]*configapi.PathValue)
			}
			config.Values[changeValue.Path] = &configapi.PathValue{
				Path:    changeValue.Path,
				Value:   changeValue.Value,
				Deleted: changeValue.Delete,
				Index:   transaction.Index,
			}
		}

		config.Status.State = configapi.ConfigurationState_CONFIGURATION_PENDING
		config.Status.TransactionIndex = transaction.Index
		err = r.configurations.Update(ctx, config)
		if err != nil {
			if !errors.IsConflict(err) && !errors.IsNotFound(err) {
				return false, err
			}
			return false, nil
		}
	}

	// Complete the transaction once the target configurations have been updated
	transaction.Status.State = configapi.TransactionState_TRANSACTION_COMPLETE
	err = r.transactions.Update(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (r *Reconciler) reconcileTransactionRollback(ctx context.Context, transaction *configapi.Transaction, rollback *configapi.TransactionRollback) (bool, error) {
	return false, nil
}
