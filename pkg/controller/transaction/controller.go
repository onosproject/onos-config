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
	if ok, err := r.reconcileTransaction(ctx, transaction); err != nil {
		log.Warnf("Failed to reconcile transaction: %s, %s", transactionID, err)
		return controller.Result{}, err
	} else if ok {
		return controller.Result{}, nil
	}

	return controller.Result{}, nil
}

func (r *Reconciler) reconcileTransaction(ctx context.Context, transaction *configapi.Transaction) (bool, error) {

	log.Debugf("Reconciling transaction %s in %s state", transaction.ID, transaction.Status.State)
	// If the transaction is Pending, begin validation if the prior transaction
	// has already been applied. This simplifies concurrency control in the controller
	// and guarantees transactions are applied to the configurations in sequential order.
	if transaction.Status.State == configapi.TransactionState_TRANSACTION_PENDING {
		if transaction.Index > 1 {
			prevTransaction, err := r.transactions.GetByIndex(ctx, transaction.Index-1)
			if err != nil {
				if !errors.IsNotFound(err) {
					return false, err
				}
				return false, nil
			}
			if prevTransaction.Status.State == configapi.TransactionState_TRANSACTION_COMPLETE ||
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
		}

	}

	//  If the transaction is in the Validating state, compute and validate the
	//  Configuration for each target.
	if transaction.Status.State == configapi.TransactionState_TRANSACTION_VALIDATING {
		for _, change := range transaction.Changes {
			modelName := utils.ToModelNameV2(change.TargetType, change.TargetVersion)
			modelPlugin, ok := r.pluginRegistry.GetPlugin(modelName)
			if !ok {
				return false, errors.NewNotFound("model plugin not found")
			}

			// If validation fails any target, mark the transaction Failed.
			// If validation is successful, proceed to Applying.
			err := modelPlugin.Validate(ctx, nil)
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

	// If the transaction is in the Applying state, update the Configuration for each
	// target and Complete the transaction.
	if transaction.Status.State == configapi.TransactionState_TRANSACTION_APPLYING {
		if transaction.Atomic {
			// TODO: Apply atomic transactions here
		} else {
			for _, transactionChange := range transaction.Changes {
				config, err := r.configurations.Get(ctx, transactionChange.TargetID)
				if err != nil {
					if !errors.IsNotFound(err) {
						return false, err
					}
					return false, nil
				}
				if config.Index < transaction.Index {
					updatedConfigValues := make(map[string]*configapi.PathValue)
					for targetID, pathValue := range config.Values {
						pathValue.Index = transaction.Index
						updatedConfigValues[targetID] = pathValue
					}
					config.Values = updatedConfigValues
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

		}
	}
	transaction.Status.State = configapi.TransactionState_TRANSACTION_COMPLETE
	err := r.transactions.Update(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}
