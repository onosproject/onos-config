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
	ID      int         `json:"id"`
	Values  ValueStates `json:"values"`
	Running bool        `json:"running"`
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

type Transactions map[Index]Transaction

func (t *Transactions) UnmarshalJSON(data []byte) error {
	var list []Transaction
	if err := json.Unmarshal(data, &list); err == nil {
		transactions := make(Transactions)
		for _, transaction := range list {
			transactions[transaction.Index] = transaction
		}
		*t = transactions
		return nil
	}
	obj := make(map[Index]Transaction)
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	transactions := make(Transactions)
	for _, transaction := range obj {
		transactions[transaction.Index] = transaction
	}
	*t = transactions
	return nil
}

type Transaction struct {
	Index    Index             `json:"index"`
	Proposal Index             `json:"proposal"`
	Type     TransactionType   `json:"type"`
	Values   Values            `json:"values"`
	Init     TransactionStatus `json:"init"`
	Commit   TransactionStatus `json:"commit"`
	Apply    TransactionStatus `json:"apply"`
}

type Values map[string]string

func (s *Values) UnmarshalJSON(data []byte) error {
	var list []any
	if err := json.Unmarshal(data, &list); err != nil {
		values := make(map[string]string)
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

type ProposalPhase string

func (p *ProposalPhase) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != nilString {
		*p = ProposalPhase(value)
	}
	return nil
}

const (
	ProposalPhaseChange   ProposalPhase = "Change"
	ProposalPhaseRollback ProposalPhase = "Rollback"
)

type ProposalStatus string

func (s *ProposalStatus) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != nilString {
		*s = ProposalStatus(value)
	}
	return nil
}

const (
	ProposalStatusPending    ProposalStatus = "Pending"
	ProposalStatusInProgress ProposalStatus = "InProgress"
	ProposalStatusComplete   ProposalStatus = "Complete"
	ProposalStatusAborted    ProposalStatus = "Aborted"
	ProposalStatusFailed     ProposalStatus = "Failed"
)

type Proposals map[Index]Proposal

func (p *Proposals) UnmarshalJSON(data []byte) error {
	var list []Proposal
	if err := json.Unmarshal(data, &list); err == nil {
		proposals := make(Proposals)
		for _, proposal := range list {
			proposals[proposal.Index] = proposal
		}
		*p = proposals
		return nil
	}
	obj := make(map[Index]Proposal)
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	proposals := make(Proposals)
	for _, proposal := range obj {
		proposals[proposal.Index] = proposal
	}
	*p = proposals
	return nil
}

type Proposal struct {
	Phase    ProposalPhase `json:"phase"`
	Index    Index         `json:"index"`
	Change   Change        `json:"change"`
	Rollback Rollback      `json:"rollback"`
}

type Change struct {
	Commit ProposalStatus `json:"commit"`
	Apply  ProposalStatus `json:"apply"`
	Values ValueStates    `json:"values"`
}

type Rollback struct {
	Commit   ProposalStatus `json:"commit"`
	Apply    ProposalStatus `json:"apply"`
	Revision Index          `json:"revision"`
	Values   ValueStates    `json:"values"`
}

type ValueStates map[string]ValueState

func (s *ValueStates) UnmarshalJSON(data []byte) error {
	var list []any
	if err := json.Unmarshal(data, &list); err != nil {
		values := make(map[string]ValueState)
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		state := make(ValueStates)
		for key, value := range values {
			state[key] = value
		}
		*s = state
	}
	return nil
}

const nilString = "<nil>"

type ValueState struct {
	Index Index `json:"index"`
	Value Value `json:"value"`
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
	Index    Index       `json:"index"`
	Revision Revision    `json:"revision"`
	Term     Term        `json:"term"`
	Values   ValueStates `json:"values"`
}

type ConfigurationApplied struct {
	Index    Index       `json:"index"`
	Revision Revision    `json:"revision"`
	Values   ValueStates `json:"values"`
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
