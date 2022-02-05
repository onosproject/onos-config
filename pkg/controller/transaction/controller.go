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

package transaction

import (
	"context"
	proposalstore "github.com/onosproject/onos-config/pkg/store/proposal"
	transactionstore "github.com/onosproject/onos-config/pkg/store/transaction"
	"time"

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
func NewController(transactions transactionstore.Store, proposals proposalstore.Store) *controller.Controller {
	c := controller.NewController("transaction")
	c.Watch(&TransactionWatcher{
		transactions: transactions,
	})
	c.Watch(&ProposalWatcher{
		proposals: proposals,
	})
	c.Reconcile(&Reconciler{
		transactions: transactions,
		proposals:    proposals,
	})
	return c
}

// Reconciler reconciles transactions
type Reconciler struct {
	transactions transactionstore.Store
	proposals    proposalstore.Store
}

// Reconcile reconciles target transactions
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
	return r.reconcileTransaction(ctx, transaction)
}

func (r *Reconciler) reconcileTransaction(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	if transaction.Status.Phases.Apply != nil {
		return r.reconcileChangeApply(ctx, transaction)
	} else if transaction.Status.Phases.Commit != nil {
		return r.reconcileChangeCommit(ctx, transaction)
	} else if transaction.Status.Phases.Validate != nil {
		return r.reconcileChangeValidate(ctx, transaction)
	} else if transaction.Status.Phases.Initialize != nil {
		return r.reconcileChangeInitialize(ctx, transaction)
	} else {
		log.Infof("Initializing Transaction %d", transaction.Index)
		transaction.Status.Phases.Initialize = &configapi.TransactionInitializePhase{
			TransactionPhaseStatus: configapi.TransactionPhaseStatus{
				Start: getCurrentTimestamp(),
			},
		}
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}
}

func (r *Reconciler) reconcileChangeInitialize(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Phases.Initialize.State {
	case configapi.TransactionInitializePhase_INITIALIZING:
		prevTransaction, err := r.transactions.GetByIndex(ctx, transaction.Index-1)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
				return controller.Result{}, err
			}
		} else {
			if prevTransaction.Status.Phases.Initialize == nil ||
				prevTransaction.Status.Phases.Initialize.State == configapi.TransactionInitializePhase_INITIALIZING {
				log.Infof("Transaction %d waiting for Transaction %d to initialize", transaction.Index, prevTransaction.Index)
				return controller.Result{}, nil
			}
		}

		if transaction.Status.Proposals == nil {
			var proposals []configapi.ProposalID
			switch details := transaction.Details.(type) {
			case *configapi.Transaction_Change:
				for targetID, change := range details.Change.Values {
					proposalID := proposalstore.NewID(targetID, transaction.Index)
					_, err := r.proposals.Get(ctx, proposalID)
					if err != nil {
						if !errors.IsNotFound(err) {
							log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
							return controller.Result{}, err
						}
						proposal := &configapi.Proposal{
							ID:               proposalID,
							TransactionIndex: transaction.Index,
							TargetID:         targetID,
							Details: &configapi.Proposal_Change{
								Change: &configapi.ChangeProposal{
									Values: change.Values,
								},
							},
						}
						err := r.proposals.Create(ctx, proposal)
						if err != nil {
							if !errors.IsAlreadyExists(err) {
								log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
								return controller.Result{}, err
							}
							return controller.Result{}, nil
						}
					}
					proposals = append(proposals, proposalID)
				}
			case *configapi.Transaction_Rollback:
				targetTransaction, err := r.transactions.GetByIndex(ctx, details.Rollback.RollbackIndex)
				if err != nil {
					if !errors.IsNotFound(err) {
						log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
						return controller.Result{}, err
					}
					err = errors.NewNotFound("transaction %d not found", details.Rollback.RollbackIndex)
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					transaction.Status.Phases.Initialize.State = configapi.TransactionInitializePhase_FAILED
					transaction.Status.Phases.Initialize.Failure = &configapi.Failure{
						Type:        configapi.Failure_NOT_FOUND,
						Description: err.Error(),
					}
					transaction.Status.Phases.Initialize.End = getCurrentTimestamp()
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				}

				switch targetDetails := targetTransaction.Details.(type) {
				case *configapi.Transaction_Change:
					for targetID := range targetDetails.Change.Values {
						proposalID := proposalstore.NewID(targetID, transaction.Index)
						_, err = r.proposals.Get(ctx, proposalID)
						if err != nil {
							if !errors.IsNotFound(err) {
								log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
								return controller.Result{}, err
							}
							proposal := &configapi.Proposal{
								ID:               proposalID,
								TransactionIndex: transaction.Index,
								TargetID:         targetID,
								Details: &configapi.Proposal_Rollback{
									Rollback: &configapi.RollbackProposal{
										RollbackIndex: details.Rollback.RollbackIndex,
									},
								},
							}
							err := r.proposals.Create(ctx, proposal)
							if err != nil {
								if !errors.IsAlreadyExists(err) {
									log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
									return controller.Result{}, err
								}
								return controller.Result{}, nil
							}
						}
						proposals = append(proposals, proposalID)
					}
				case *configapi.Transaction_Rollback:
					err = errors.NewNotFound("transaction %d is not a valid change", details.Rollback.RollbackIndex)
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					transaction.Status.Phases.Initialize.State = configapi.TransactionInitializePhase_FAILED
					transaction.Status.Phases.Initialize.Failure = &configapi.Failure{
						Type:        configapi.Failure_FORBIDDEN,
						Description: err.Error(),
					}
					transaction.Status.Phases.Initialize.End = getCurrentTimestamp()
					if err := r.updateTransactionStatus(ctx, transaction); err != nil {
						return controller.Result{}, err
					}
					return controller.Result{}, nil
				}
			}
			transaction.Status.Proposals = proposals
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}

		allInitialized := true
		for _, proposalID := range transaction.Status.Proposals {
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			if proposal.Status.Phases.Initialize == nil {
				log.Infof("Initializing Transaction %d changes to target '%s'", transaction.Index, proposal.TargetID)
				proposal.Status.Phases.Initialize = &configapi.ProposalInitializePhase{
					ProposalPhaseStatus: configapi.ProposalPhaseStatus{
						Start: getCurrentTimestamp(),
					},
				}
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			switch proposal.Status.Phases.Initialize.State {
			case configapi.ProposalInitializePhase_INITIALIZING:
				allInitialized = false
			}
		}

		if allInitialized {
			log.Infof("Transaction %d initialized", transaction.Index)
			transaction.Status.Phases.Initialize.State = configapi.TransactionInitializePhase_INITIALIZED
			transaction.Status.Phases.Initialize.End = getCurrentTimestamp()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	case configapi.TransactionInitializePhase_INITIALIZED:
		checked := make(map[configapi.Index]bool)
		for _, proposalID := range transaction.Status.Proposals {
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}
			if proposal.Status.PrevIndex > 0 {
				if _, ok := checked[proposal.Status.PrevIndex]; !ok {
					prevTransaction, err := r.transactions.GetByIndex(ctx, proposal.Status.PrevIndex)
					if err != nil {
						if !errors.IsNotFound(err) {
							return controller.Result{}, err
						}
					} else {
						// Return if waiting for the previous transaction to commit.
						if !(prevTransaction.Status.Phases.Validate != nil && prevTransaction.Status.Phases.Validate.State == configapi.TransactionValidatePhase_FAILED) &&
							!(prevTransaction.Status.Phases.Commit != nil && prevTransaction.Status.Phases.Commit.State == configapi.TransactionCommitPhase_COMMITTED) {
							log.Infof("Transaction %d waiting for Transaction %d to be committed", transaction.Index, prevTransaction.Index)
							return controller.Result{}, nil
						}
					}
					checked[proposal.Status.PrevIndex] = true
				}
			}
		}

		log.Infof("Validating Transaction %d", transaction.Index)
		transaction.Status.Phases.Validate = &configapi.TransactionValidatePhase{
			TransactionPhaseStatus: configapi.TransactionPhaseStatus{
				Start: getCurrentTimestamp(),
			},
		}
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{
			Requeue: controller.NewID(transaction.Index + 1),
		}, nil
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) reconcileChangeValidate(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Phases.Validate.State {
	case configapi.TransactionValidatePhase_VALIDATING:
		allValidated := true
		for _, proposalID := range transaction.Status.Proposals {
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			if proposal.Status.Phases.Validate == nil {
				log.Infof("Validating Transaction %d changes to target '%s'", transaction.Index, proposal.TargetID)
				proposal.Status.Phases.Validate = &configapi.ProposalValidatePhase{
					ProposalPhaseStatus: configapi.ProposalPhaseStatus{
						Start: getCurrentTimestamp(),
					},
				}
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			switch proposal.Status.Phases.Validate.State {
			case configapi.ProposalValidatePhase_VALIDATING:
				allValidated = false
			case configapi.ProposalValidatePhase_FAILED:
				log.Infof("Transaction %d failed", transaction.Index)
				transaction.Status.Phases.Validate.State = configapi.TransactionValidatePhase_FAILED
				transaction.Status.Phases.Validate.End = getCurrentTimestamp()
				if err := r.updateTransactionStatus(ctx, transaction); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}
		}

		if allValidated {
			log.Infof("Transaction %d committed", transaction.Index)
			transaction.Status.Phases.Validate.State = configapi.TransactionValidatePhase_VALIDATED
			transaction.Status.Phases.Validate.End = getCurrentTimestamp()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	case configapi.TransactionValidatePhase_VALIDATED:
		checked := make(map[configapi.Index]bool)
		for _, proposalID := range transaction.Status.Proposals {
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}
			if proposal.Status.PrevIndex > 0 {
				if _, ok := checked[proposal.Status.PrevIndex]; !ok {
					prevTransaction, err := r.transactions.GetByIndex(ctx, proposal.Status.PrevIndex)
					if err != nil {
						if !errors.IsNotFound(err) {
							return controller.Result{}, err
						}
					} else {
						// Return if waiting for the previous transaction to commit.
						if !(prevTransaction.Status.Phases.Validate != nil && prevTransaction.Status.Phases.Validate.State == configapi.TransactionValidatePhase_FAILED) &&
							!(prevTransaction.Status.Phases.Commit != nil && prevTransaction.Status.Phases.Commit.State == configapi.TransactionCommitPhase_COMMITTED) {
							log.Infof("Transaction %d waiting for Transaction %d to be committed", transaction.Index, prevTransaction.Index)
							return controller.Result{}, nil
						}
					}
					checked[proposal.Status.PrevIndex] = true
				}
			}
		}

		log.Infof("Committing Transaction %d", transaction.Index)
		transaction.Status.Phases.Commit = &configapi.TransactionCommitPhase{
			TransactionPhaseStatus: configapi.TransactionPhaseStatus{
				Start: getCurrentTimestamp(),
			},
		}
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{
			Requeue: controller.NewID(transaction.Index + 1),
		}, nil
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) reconcileChangeCommit(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Phases.Commit.State {
	case configapi.TransactionCommitPhase_COMMITTING:
		allCommitted := true
		for _, proposalID := range transaction.Status.Proposals {
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			if proposal.Status.Phases.Commit == nil {
				log.Infof("Committing Transaction %d changes to target '%s'", transaction.Index, proposal.TargetID)
				proposal.Status.Phases.Commit = &configapi.ProposalCommitPhase{
					ProposalPhaseStatus: configapi.ProposalPhaseStatus{
						Start: getCurrentTimestamp(),
					},
				}
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			switch proposal.Status.Phases.Commit.State {
			case configapi.ProposalCommitPhase_COMMITTING:
				allCommitted = false
			}
		}

		if allCommitted {
			log.Infof("Transaction %d committed", transaction.Index)
			transaction.Status.Phases.Commit.State = configapi.TransactionCommitPhase_COMMITTED
			transaction.Status.Phases.Commit.End = getCurrentTimestamp()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	case configapi.TransactionCommitPhase_COMMITTED:
		checked := make(map[configapi.Index]bool)
		for _, proposalID := range transaction.Status.Proposals {
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}
			if proposal.Status.PrevIndex > 0 {
				if _, ok := checked[proposal.Status.PrevIndex]; !ok {
					prevTransaction, err := r.transactions.GetByIndex(ctx, proposal.Status.PrevIndex)
					if err != nil {
						if !errors.IsNotFound(err) {
							return controller.Result{}, err
						}
					} else {
						// Return if waiting for the previous transaction to be applied.
						if !(prevTransaction.Status.Phases.Validate != nil && prevTransaction.Status.Phases.Validate.State == configapi.TransactionValidatePhase_FAILED) &&
							!(prevTransaction.Status.Phases.Apply != nil && prevTransaction.Status.Phases.Apply.State == configapi.TransactionApplyPhase_APPLIED) {
							log.Infof("Transaction %d waiting for Transaction %d to be applied", transaction.Index, prevTransaction.Index)
							return controller.Result{}, nil
						}
					}
					checked[proposal.Status.PrevIndex] = true
				}
			}
		}

		log.Infof("Applying Transaction %d", transaction.Index)
		transaction.Status.Phases.Apply = &configapi.TransactionApplyPhase{
			TransactionPhaseStatus: configapi.TransactionPhaseStatus{
				Start: getCurrentTimestamp(),
			},
		}
		if err := r.updateTransactionStatus(ctx, transaction); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{
			Requeue: controller.NewID(transaction.Index + 1),
		}, nil
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) reconcileChangeApply(ctx context.Context, transaction *configapi.Transaction) (controller.Result, error) {
	switch transaction.Status.Phases.Apply.State {
	case configapi.TransactionApplyPhase_APPLYING:
		allApplied := true
		for _, proposalID := range transaction.Status.Proposals {
			proposal, err := r.proposals.Get(ctx, proposalID)
			if err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed reconciling Transaction %d", transaction.Index, err)
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			if proposal.Status.Phases.Apply == nil {
				log.Infof("Applying Transaction %d changes to target '%s'", transaction.Index, proposal.TargetID)
				proposal.Status.Phases.Apply = &configapi.ProposalApplyPhase{
					ProposalPhaseStatus: configapi.ProposalPhaseStatus{
						Start: getCurrentTimestamp(),
					},
				}
				if err := r.updateProposalStatus(ctx, proposal); err != nil {
					return controller.Result{}, err
				}
				return controller.Result{}, nil
			}

			switch proposal.Status.Phases.Apply.State {
			case configapi.ProposalApplyPhase_APPLYING:
				allApplied = false
			}
		}

		if allApplied {
			log.Infof("Transaction %d applied", transaction.Index)
			transaction.Status.Phases.Apply.State = configapi.TransactionApplyPhase_APPLIED
			transaction.Status.Phases.Apply.End = getCurrentTimestamp()
			if err := r.updateTransactionStatus(ctx, transaction); err != nil {
				return controller.Result{}, err
			}
			return controller.Result{}, nil
		}
		return controller.Result{}, nil
	case configapi.TransactionApplyPhase_APPLIED:
		return controller.Result{
			Requeue: controller.NewID(transaction.Index + 1),
		}, nil
	default:
		return controller.Result{}, nil
	}
}

func (r *Reconciler) updateTransactionStatus(ctx context.Context, transaction *configapi.Transaction) error {
	log.Debug(transaction.Status)
	err := r.transactions.UpdateStatus(ctx, transaction)
	if err != nil {
		if !errors.IsNotFound(err) && !errors.IsConflict(err) {
			log.Errorf("Failed updating Transaction %d status", transaction.Index, err)
			return err
		}
		log.Warnf("Write conflict updating Transaction %d status", transaction.Index, err)
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
