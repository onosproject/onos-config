package model

import (
	"encoding/json"
)

type TestCase[S, C any] struct {
	Context     C `json:"context"`
	State       S `json:"state"`
	Transitions S `json:"transitions"`
}

type Index uint64

type Revision uint64

type Term uint64

type NodeID string

type Conn struct {
	ID        int  `json:"id"`
	Connected bool `json:"connected"`
}

type Target struct {
	ID      int    `json:"id"`
	Values  Values `json:"values"`
	Running bool   `json:"running"`
}

type TransactionType string

const (
	TransactionTypeChange   TransactionType = "Change"
	TransactionTypeRollback TransactionType = "Rollback"
)

type TransactionStatus string

const (
	TransactionStatusPending    TransactionStatus = "Pending"
	TransactionStatusInProgress TransactionStatus = "InProgress"
	TransactionStatusComplete   TransactionStatus = "Complete"
	TransactionStatusAborted    TransactionStatus = "Aborted"
	TransactionStatusFailed     TransactionStatus = "Failed"
)

type Transaction struct {
	Index    Index             `json:"index"`
	Proposal Index             `json:"proposal"`
	Type     TransactionType   `json:"type"`
	Values   map[string]string `json:"values"`
	Init     TransactionStatus `json:"init"`
	Commit   TransactionStatus `json:"commit"`
	Apply    TransactionStatus `json:"apply"`
}

type ProposalPhase string

const (
	ProposalPhaseCommit ProposalPhase = "Commit"
	ProposalPhaseApply  ProposalPhase = "Apply"
)

type ProposalStatus string

const (
	ProposalStatusPending    ProposalStatus = "Pending"
	ProposalStatusInProgress ProposalStatus = "InProgress"
	ProposalStatusComplete   ProposalStatus = "Complete"
	ProposalStatusAborted    ProposalStatus = "Aborted"
	ProposalStatusFailed     ProposalStatus = "Failed"
)

type Proposal struct {
	Index    Index    `json:"index"`
	Change   Change   `json:"change"`
	Rollback Rollback `json:"rollback"`
}

type Change struct {
	Phase  ProposalPhase  `json:"phase"`
	Status ProposalStatus `json:"state"`
	Values Values         `json:"values"`
}

type Rollback struct {
	Phase    ProposalPhase  `json:"phase"`
	Status   ProposalStatus `json:"state"`
	Revision Index          `json:"revision"`
	Values   Values         `json:"values"`
}

type Values map[string]ValueState

func (s *Values) UnmarshalJSON(data []byte) error {
	var list []any
	if err := json.Unmarshal(data, &list); err != nil {
		values := make(map[string]ValueState)
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		state := make(Values)
		for key, value := range values {
			state[key] = value
		}
		*s = state
	}
	return nil
}

const nilString = "<nil>"

type ValueState struct {
	Index Index  `json:"index"`
	Value *Value `json:"value"`
}

type Value string

func (s *Value) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != nilString {
		*s = Value(value)
	}
	return nil
}

type Configuration struct {
	Status    ConfigurationStatus    `json:"state"`
	Term      Term                   `json:"term"`
	Committed ConfigurationCommitted `json:"committed"`
	Applied   ConfigurationApplied   `json:"applied"`
}

type ConfigurationCommitted struct {
	Index    Index    `json:"index"`
	Revision Revision `json:"revision"`
	Term     Term     `json:"term"`
	Values   Values   `json:"values"`
}

type ConfigurationApplied struct {
	Index    Index    `json:"index"`
	Revision Revision `json:"revision"`
	Values   Values   `json:"values"`
}

type ConfigurationStatus string

const (
	ConfigurationStatusInProgress ConfigurationStatus = "InProgress"
	ConfigurationStatusComplete   ConfigurationStatus = "Complete"
)

type Mastership struct {
	Term   Term   `json:"term"`
	Master NodeID `json:"master"`
}
