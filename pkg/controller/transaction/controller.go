// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package transaction

import (
	"context"
	configurationstore "github.com/onosproject/onos-config/pkg/store/configuration"
	"time"

	proposalstore "github.com/onosproject/onos-config/pkg/store/proposal"
	transactionstore "github.com/onosproject/onos-config/pkg/store/transaction"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "transaction")

const (
	defaultTimeout = 30 * time.Second
)

// NewController returns a transaction controller
func NewController(transactions transactionstore.Store, proposals proposalstore.Store, configurations configurationstore.Store) *controller.Controller {
	c := controller.NewController("transaction")
	c.Watch(&Watcher{
		transactions: transactions,
	})
	c.Watch(&ProposalWatcher{
		proposals: proposals,
	})
	c.Reconcile(&Reconciler{
		transactions:   transactions,
		proposals:      proposals,
		configurations: configurations,
	})
	return c
}

// Reconciler reconciles transactions
type Reconciler struct {
	transactions   transactionstore.Store
	proposals      proposalstore.Store
	configurations configurationstore.Store
}

// Reconcile reconciles target transactions
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	transactionID := id.Value.(configapi.TransactionID)
	transaction, err := r.transactions.Get(ctx, transactionID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Warnf("Failed to reconcile Transaction %s", transactionID, err)
			return controller.Result{}, err
		}
		log.Debugf("Transaction %s not found", transactionID)
		return controller.Result{}, nil
	}

	log.Debugf("Reconciling Transaction %s", transactionID)
	log.Debug(transaction)
	return r.reconcileTransaction(ctx, transaction)
}

func isComplete(state configapi.TransactionPhaseStatus_State) bool {
	return state >= configapi.TransactionPhaseStatus_COMPLETE
}

func (r *Reconciler) reconcileTransaction(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	if !isComplete(transaction.Status.Initialize.State) {
		return r.reconcileInitialize(ctx, transaction)
	}
	if !isComplete(transaction.Status.Commit.State) {
		return r.reconcileCommit(ctx, transaction)
	}
	if !isComplete(transaction.Status.Apply.State) {
		return r.reconcileApply(ctx, transaction)
	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcileInitialize(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	prevTransactionID := configapi.TransactionID{
		Target: transaction.ID.Target,
		Index:  transaction.ID.Index - 1,
	}
	prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}
	} else {
		if !isComplete(prevTransaction.Status.Initialize.State) {
			log.Infof("Transaction %s waiting for Transaction %s to initialize", transaction.ID, prevTransactionID)
			return controller.Result{}, nil
		}
	}

	switch transaction.Details.(type) {
	case *configapi.Transaction_Change:
		return r.initializeChange(ctx, transaction)
	case *configapi.Transaction_Rollback:
		return r.initializeRollback(ctx, transaction)
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) initializeChange(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	_, err := r.proposals.GetKey(ctx, transaction.ID.Target, transaction.Key)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}

		// Extract type/version override for this target, if available
		changeValues := transaction.GetChange().Values
		changeValuesWithIndex := make(map[string]*configapi.PathValue)
		for targetID, changeValue := range changeValues {
			changeValue.Index = transaction.ID.Index
			changeValuesWithIndex[targetID] = changeValue
		}

		// Create a proposal for this target
		proposal := &configapi.Proposal{
			ObjectMeta: configapi.ObjectMeta{
				Key: transaction.Key,
			},
			ID: configapi.ProposalID{
				Target: transaction.ID.Target,
			},
			Values: changeValuesWithIndex,
			Status: configapi.ProposalStatus{
				Change: configapi.ProposalChangeStatus{
					Transaction: transaction.ID.Index,
					Commit: configapi.ProposalPhaseStatus{
						State: configapi.ProposalPhaseStatus_PENDING,
					},
					Apply: configapi.ProposalPhaseStatus{
						State: configapi.ProposalPhaseStatus_PENDING,
					},
				},
			},
		}
		err := r.proposals.Create(ctx, proposal)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	}

	transaction.Status.Initialize.State = configapi.TransactionPhaseStatus_COMPLETE
	transaction.Status.Initialize.End = getCurrentTimestamp()
	transaction.Status.State = configapi.TransactionStatus_PENDING
	if err := r.updateTransactionStatus(ctx, transaction); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

func (r *Reconciler) initializeRollback(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	targetProposalID := configapi.ProposalID{
		Target: transaction.ID.Target,
		Index:  transaction.GetRollback().Index,
	}
	_, err := r.proposals.Get(ctx, targetProposalID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}
		err = errors.NewNotFound("proposal %s not found", targetProposalID)
		log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
		failure := &configapi.Failure{
			Type:        configapi.Failure_NOT_FOUND,
			Description: err.Error(),
		}
		transaction.Status.Initialize.State = configapi.TransactionPhaseStatus_FAILED
		transaction.Status.Initialize.Failure = failure
		transaction.Status.Initialize.End = getCurrentTimestamp()
		transaction.Status.State = configapi.TransactionStatus_FAILED
		transaction.Status.Failure = failure
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	nextProposalID := configapi.ProposalID{
		Target: transaction.ID.Target,
		Index:  transaction.GetRollback().Index + 1,
	}
	nextProposal, err := r.proposals.Get(ctx, nextProposalID)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}
	} else if nextProposal.Status.Rollback.Commit.State == configapi.ProposalPhaseStatus_INACTIVE {
		err = errors.NewForbidden("subsequent proposals must be rolled back before %s can be rolled back", targetProposalID)
		log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
		failure := &configapi.Failure{
			Type:        configapi.Failure_FORBIDDEN,
			Description: err.Error(),
		}
		transaction.Status.Initialize.State = configapi.TransactionPhaseStatus_FAILED
		transaction.Status.Initialize.Failure = failure
		transaction.Status.Initialize.End = getCurrentTimestamp()
		transaction.Status.State = configapi.TransactionStatus_FAILED
		transaction.Status.Failure = failure
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	transaction.Status.Initialize.State = configapi.TransactionPhaseStatus_COMPLETE
	transaction.Status.Initialize.End = getCurrentTimestamp()
	transaction.Status.State = configapi.TransactionStatus_PENDING
	if err := r.updateTransactionStatus(ctx, transaction); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcileCommit(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Commit.State {
	case configapi.TransactionPhaseStatus_PENDING:
		prevTransactionID := configapi.TransactionID{
			Target: transaction.ID.Target,
			Index:  transaction.ID.Index - 1,
		}
		prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
				return controller.Result{}, err
			}
		} else {
			if !isComplete(prevTransaction.Status.Commit.State) {
				log.Infof("Transaction %s waiting for Transaction %s to be committed", transaction.ID, prevTransactionID)
				return controller.Result{}, nil
			}
		}
	}

	switch transaction.Details.(type) {
	case *configapi.Transaction_Change:
		return r.commitChange(ctx, transaction)
	case *configapi.Transaction_Rollback:
		return r.commitRollback(ctx, transaction)
	}
	return controller.Result{}, nil
}

func (r *Reconciler) commitChange(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Commit.State {
	case configapi.TransactionPhaseStatus_PENDING:
		transaction.Status.Commit.Start = getCurrentTimestamp()
		transaction.Status.Commit.State = configapi.TransactionPhaseStatus_IN_PROGRESS
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, err
		}
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		proposal, err := r.proposals.GetKey(ctx, transaction.ID.Target, transaction.Key)
		if err != nil {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}

		switch proposal.Status.Change.Commit.State {
		case configapi.ProposalPhaseStatus_PENDING:
			configurationID := configapi.ConfigurationID{
				Target: proposal.ID.Target,
			}
			configuration, err := r.configurations.Get(ctx, configurationID)

			var committedIndex configapi.Index
			var committedValues map[string]*configapi.PathValue
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
					return controller.Result{}, err
				}
				committedValues = make(map[string]*configapi.PathValue)
			} else {
				committedIndex = configuration.Index
				committedValues = configuration.Values
			}

			rollbackValues := make(map[string]*configapi.PathValue)
			for path := range proposal.Values {
				if committedValue, ok := committedValues[path]; ok {
					rollbackValues[path] = committedValue
				} else {
					rollbackValues[path] = &configapi.PathValue{
						Path:    path,
						Deleted: true,
					}
				}
			}

			proposal.Status.Change.Commit.Start = getCurrentTimestamp()
			proposal.Status.Change.Commit.State = configapi.ProposalPhaseStatus_IN_PROGRESS
			proposal.Status.Rollback = configapi.ProposalRollbackStatus{
				Index:  committedIndex,
				Values: rollbackValues,
			}
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_COMPLETE:
			transaction.Status.Commit.State = configapi.TransactionPhaseStatus_COMPLETE
			transaction.Status.Commit.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_PENDING
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_FAILED:
			err = errors.NewInvalid("proposal %s is invalid", proposal.ID)
			log.Warnf("Failed reconciling Transaction %s", transaction.ID, err)
			failure := &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: err.Error(),
			}
			transaction.Status.Commit.State = configapi.TransactionPhaseStatus_FAILED
			transaction.Status.Commit.Failure = failure
			transaction.Status.Commit.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_FAILED
			transaction.Status.Failure = failure
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	}
	return controller.Result{}, nil
}

func (r *Reconciler) commitRollback(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Commit.State {
	case configapi.TransactionPhaseStatus_PENDING:
		switch transaction.Status.Initialize.State {
		case configapi.TransactionPhaseStatus_COMPLETE:
			proposalID := configapi.ProposalID{
				Target: transaction.ID.Target,
				Index:  transaction.GetRollback().Index,
			}
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
				return controller.Result{}, err
			}

			if proposal.Status.Rollback.Commit.State == configapi.ProposalPhaseStatus_INACTIVE {
				switch proposal.Status.Change.Commit.State {
				case configapi.ProposalPhaseStatus_PENDING:
					proposal.Status.Change.Commit.State = configapi.ProposalPhaseStatus_ABORTED
					proposal.Status.Change.Commit.End = getCurrentTimestamp()
					if err := r.updateProposalStatus(ctx, proposal); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				case configapi.ProposalPhaseStatus_COMPLETE:
					proposal.Status.Rollback.Transaction = transaction.ID.Index
					proposal.Status.Rollback.Commit.State = configapi.ProposalPhaseStatus_PENDING
					proposal.Status.Rollback.Apply.State = configapi.ProposalPhaseStatus_PENDING
					if err := r.updateProposalStatus(ctx, proposal); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				case configapi.ProposalPhaseStatus_ABORTED, configapi.ProposalPhaseStatus_FAILED:
					transaction.Status.Commit.Start = getCurrentTimestamp()
					transaction.Status.Commit.State = configapi.TransactionPhaseStatus_COMPLETE
					transaction.Status.Commit.End = getCurrentTimestamp()
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				}
			} else {
				switch proposal.Status.Rollback.Commit.State {
				case configapi.ProposalPhaseStatus_PENDING:
					transaction.Status.Commit.Start = getCurrentTimestamp()
					transaction.Status.Commit.State = configapi.TransactionPhaseStatus_IN_PROGRESS
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				case configapi.ProposalPhaseStatus_ABORTED:
					transaction.Status.Commit.Start = getCurrentTimestamp()
					transaction.Status.Commit.State = configapi.TransactionPhaseStatus_ABORTED
					transaction.Status.Commit.End = getCurrentTimestamp()
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				}
			}
		case configapi.TransactionPhaseStatus_FAILED:
			transaction.Status.Commit.Start = getCurrentTimestamp()
			transaction.Status.Commit.End = getCurrentTimestamp()
			transaction.Status.Commit.State = configapi.TransactionPhaseStatus_ABORTED
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		proposal, err := r.proposals.GetKey(ctx, transaction.ID.Target, transaction.Key)
		if err != nil {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}

		switch proposal.Status.Rollback.Commit.State {
		case configapi.ProposalPhaseStatus_PENDING:
			proposal.Status.Rollback.Commit.Start = getCurrentTimestamp()
			proposal.Status.Rollback.Commit.State = configapi.ProposalPhaseStatus_IN_PROGRESS
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_COMPLETE:
			transaction.Status.Commit.State = configapi.TransactionPhaseStatus_COMPLETE
			transaction.Status.Commit.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_PENDING
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_FAILED:
			err = errors.NewInvalid("proposal %s is invalid", proposal.ID)
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			failure := &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: err.Error(),
			}
			transaction.Status.Commit.State = configapi.TransactionPhaseStatus_FAILED
			transaction.Status.Commit.Failure = failure
			transaction.Status.Commit.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_FAILED
			transaction.Status.Failure = failure
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcileApply(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Apply.State {
	case configapi.TransactionPhaseStatus_PENDING:
		prevTransactionID := configapi.TransactionID{
			Target: transaction.ID.Target,
			Index:  transaction.ID.Index - 1,
		}
		prevTransaction, err := r.transactions.Get(ctx, prevTransactionID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
				return controller.Result{}, err
			}
		} else {
			if !isComplete(prevTransaction.Status.Commit.State) {
				log.Infof("Transaction %s waiting for Transaction %s to be applied", transaction.ID, prevTransactionID)
				return controller.Result{}, nil
			}
		}
	}

	switch transaction.Details.(type) {
	case *configapi.Transaction_Change:
		return r.applyChange(ctx, transaction)
	case *configapi.Transaction_Rollback:
		return r.applyRollback(ctx, transaction)
	}
	return controller.Result{}, nil
}

func (r *Reconciler) applyChange(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Apply.State {
	case configapi.TransactionPhaseStatus_PENDING:
		prevProposal, err := r.proposals.GetKey(ctx, transaction.ID.Target, transaction.Key)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
				return controller.Result{}, err
			}
		} else {
			if prevProposal.Status.Change.Apply.State == configapi.ProposalPhaseStatus_FAILED &&
				prevProposal.Status.Rollback.Apply.State != configapi.ProposalPhaseStatus_COMPLETE {
				log.Infof("Transaction %s cannot be applied until failed Proposal %s is rolled back", transaction.ID, prevProposal.ID)
				return controller.Result{}, nil
			}
		}

		switch transaction.Status.Commit.State {
		case configapi.TransactionPhaseStatus_COMPLETE:
			transaction.Status.Apply.Start = getCurrentTimestamp()
			transaction.Status.Apply.State = configapi.TransactionPhaseStatus_IN_PROGRESS
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
		case configapi.TransactionPhaseStatus_FAILED:
			transaction.Status.Apply.Start = getCurrentTimestamp()
			transaction.Status.Apply.State = configapi.TransactionPhaseStatus_ABORTED
			transaction.Status.Apply.End = getCurrentTimestamp()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		proposal, err := r.proposals.GetKey(ctx, transaction.ID.Target, transaction.Key)
		if err != nil {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}

		switch proposal.Status.Change.Apply.State {
		case configapi.ProposalPhaseStatus_PENDING:
			proposal.Status.Change.Apply.Start = getCurrentTimestamp()
			proposal.Status.Change.Apply.State = configapi.ProposalPhaseStatus_IN_PROGRESS
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_COMPLETE:
			transaction.Status.Apply.State = configapi.TransactionPhaseStatus_COMPLETE
			transaction.Status.Apply.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_PENDING
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_FAILED:
			err = errors.NewInvalid("proposal %s is invalid", proposal.ID)
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			failure := &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: err.Error(),
			}
			transaction.Status.Apply.State = configapi.TransactionPhaseStatus_FAILED
			transaction.Status.Apply.Failure = failure
			transaction.Status.Apply.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_FAILED
			transaction.Status.Failure = failure
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	}
	return controller.Result{}, nil
}

func (r *Reconciler) applyRollback(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Apply.State {
	case configapi.TransactionPhaseStatus_PENDING:
		switch transaction.Status.Initialize.State {
		case configapi.TransactionPhaseStatus_COMPLETE:
			proposalID := configapi.ProposalID{
				Target: transaction.ID.Target,
				Index:  transaction.GetRollback().Index,
			}
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
				return controller.Result{}, err
			}

			if proposal.Status.Rollback.Commit.State == configapi.ProposalPhaseStatus_INACTIVE {
				switch proposal.Status.Change.Apply.State {
				case configapi.ProposalPhaseStatus_PENDING:
					proposal.Status.Change.Apply.State = configapi.ProposalPhaseStatus_ABORTED
					proposal.Status.Change.Apply.End = getCurrentTimestamp()
					if err := r.updateProposalStatus(ctx, proposal); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				case configapi.ProposalPhaseStatus_COMPLETE:
					proposal.Status.Rollback.Apply.State = configapi.ProposalPhaseStatus_PENDING
					if err := r.updateProposalStatus(ctx, proposal); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				case configapi.ProposalPhaseStatus_ABORTED, configapi.ProposalPhaseStatus_FAILED:
					transaction.Status.Apply.Start = getCurrentTimestamp()
					transaction.Status.Apply.State = configapi.TransactionPhaseStatus_COMPLETE
					transaction.Status.Apply.End = getCurrentTimestamp()
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				}
			} else {
				switch proposal.Status.Rollback.Apply.State {
				case configapi.ProposalPhaseStatus_PENDING:
					transaction.Status.Apply.Start = getCurrentTimestamp()
					transaction.Status.Apply.State = configapi.TransactionPhaseStatus_IN_PROGRESS
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				case configapi.ProposalPhaseStatus_ABORTED:
					transaction.Status.Apply.Start = getCurrentTimestamp()
					transaction.Status.Apply.State = configapi.TransactionPhaseStatus_ABORTED
					transaction.Status.Apply.End = getCurrentTimestamp()
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				}
			}
		case configapi.TransactionPhaseStatus_FAILED:
			transaction.Status.Apply.Start = getCurrentTimestamp()
			transaction.Status.Apply.End = getCurrentTimestamp()
			transaction.Status.Apply.State = configapi.TransactionPhaseStatus_ABORTED
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	case configapi.TransactionPhaseStatus_IN_PROGRESS:
		proposal, err := r.proposals.GetKey(ctx, transaction.ID.Target, transaction.Key)
		if err != nil {
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			return controller.Result{}, err
		}

		switch proposal.Status.Rollback.Apply.State {
		case configapi.ProposalPhaseStatus_PENDING:
			proposal.Status.Rollback.Apply.Start = getCurrentTimestamp()
			proposal.Status.Rollback.Apply.State = configapi.ProposalPhaseStatus_IN_PROGRESS
			if err := r.updateProposalStatus(ctx, proposal); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_COMPLETE:
			transaction.Status.Apply.State = configapi.TransactionPhaseStatus_COMPLETE
			transaction.Status.Apply.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_PENDING
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		case configapi.ProposalPhaseStatus_FAILED:
			err = errors.NewInvalid("proposal %s is invalid", proposal.ID)
			log.Errorf("Failed reconciling Transaction %s", transaction.ID, err)
			failure := &configapi.Failure{
				Type:        configapi.Failure_INVALID,
				Description: err.Error(),
			}
			transaction.Status.Apply.State = configapi.TransactionPhaseStatus_FAILED
			transaction.Status.Apply.Failure = failure
			transaction.Status.Apply.End = getCurrentTimestamp()
			transaction.Status.State = configapi.TransactionStatus_FAILED
			transaction.Status.Failure = failure
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
	}
	return controller.Result{}, nil
}

func (r *Reconciler) updateTransactionStatus(ctx context.Context, transaction *configapi.Transaction) error {
	log.Debug(transaction.Status)
	err := r.transactions.UpdateStatus(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Transaction %s status", transaction.ID, err)
			return err
		}
		log.Warnf("Write conflict updating Transaction %s status", transaction.ID, err)
		return nil
	}
	return nil
}

func (r *Reconciler) updateProposalStatus(ctx context.Context, proposal *configapi.Proposal) error {
	log.Debug(proposal.Status)
	err := r.proposals.UpdateStatus(ctx, proposal)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Proposal '%s' status", proposal.ID, err)
			return err
		}
		log.Warnf("Write conflict updating Proposal '%s' status", proposal.ID, err)
		return nil
	}
	return nil
}

func getCurrentTimestamp() *time.Time {
	t := time.Now()
	return &t
}
