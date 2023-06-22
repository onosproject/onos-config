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

type Ordinal uint64

type Revision uint64

type Term uint64

type NodeID string

type ConnID int

type Conns map[NodeID]Conn

type Conn struct {
	ID        ConnID `json:"id"`
	Connected bool   `json:"connected"`
}

type Target struct {
	ID      int    `json:"id"`
	Values  Values `json:"values"`
	Running bool   `json:"running"`
}

type EventType string

const (
	CommitEvent EventType = "Commit"
	ApplyEvent  EventType = "Apply"
)

type TransactionEvent struct {
	Index  Index              `json:"index"`
	Phase  TransactionPhaseID `json:"phase"`
	Event  EventType          `json:"event"`
	Status TransactionState   `json:"status"`
}

type TransactionPhaseID string

const (
	TransactionPhaseChange   TransactionPhaseID = "Change"
	TransactionPhaseRollback TransactionPhaseID = "Rollback"
)

type TransactionState string

func (s *TransactionState) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != nilString {
		*s = TransactionState(value)
	}
	return nil
}

const (
	TransactionPending    TransactionState = "Pending"
	TransactionInProgress TransactionState = "InProgress"
	TransactionComplete   TransactionState = "Complete"
	TransactionAborted    TransactionState = "Aborted"
	TransactionCanceled   TransactionState = "Canceled"
	TransactionFailed     TransactionState = "Failed"
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
	Index    Index              `json:"index"`
	Phase    TransactionPhaseID `json:"phase"`
	Change   TransactionPhase   `json:"change"`
	Rollback TransactionPhase   `json:"rollback"`
}

type TransactionPhase struct {
	Index   Index             `json:"index"`
	Ordinal Ordinal           `json:"ordinal"`
	Values  Values            `json:"values"`
	Commit  *TransactionState `json:"commit"`
	Apply   *TransactionState `json:"apply"`
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

const nilString = "<nil>"

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
	State     ConfigurationState     `json:"state"`
	Term      Term                   `json:"term"`
	Committed ConfigurationCommitted `json:"committed"`
	Applied   ConfigurationApplied   `json:"applied"`
}

type ConfigurationCommitted struct {
	Index    Index    `json:"index"`
	Change   Index    `json:"change"`
	Ordinal  Ordinal  `json:"ordinal"`
	Revision Revision `json:"revision"`
	Target   Index    `json:"target"`
	Term     Term     `json:"term"`
	Values   Values   `json:"values"`
}

type ConfigurationApplied struct {
	Index    Index    `json:"index"`
	Ordinal  Ordinal  `json:"ordinal"`
	Revision Revision `json:"revision"`
	Target   Index    `json:"target"`
	Values   Values   `json:"values"`
}

type ConfigurationState string

const (
	ConfigurationPending  ConfigurationState = "Pending"
	ConfigurationComplete ConfigurationState = "Complete"
)

type Mastership struct {
	Term   Term   `json:"term"`
	Master NodeID `json:"master"`
}
