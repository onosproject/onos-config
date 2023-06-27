// SPDX-FileCopyrightText: 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"encoding/json"
)

const nilString = "<nil>"

// Index is the position of a transaction in the transaction log
type Index uint64

// Ordinal indicates the order in which a transaction is applied or rolled back
type Ordinal uint64

// Revision is the current version of an object
type Revision uint64

// Term is a mastership term
type Term uint64

// NodeID is a unique node identifier
type NodeID string

// ConnID is a unique connection identifier
type ConnID int

// Conns is a mapping of node IDs to connections
type Conns map[NodeID]Conn

// Conn is the state of the connection for a node
type Conn struct {
	ID        ConnID `json:"id"`
	Connected bool   `json:"connected"`
}

// Target is the state of a target
type Target struct {
	ID      int    `json:"id"`
	Values  Values `json:"values"`
	Running bool   `json:"running"`
}

// EventType is a transaction event type constant
type EventType string

const (
	// CommitEvent indicates a transaction was committed
	CommitEvent EventType = "Commit"
	// ApplyEvent indicates a transaction was applied
	ApplyEvent EventType = "Apply"
)

// TransactionEvent indicates the latest event within a transaction
type TransactionEvent struct {
	// Index is the transaction index
	Index Index `json:"index"`
	// Phase is the phase in which the event occurred
	Phase TransactionPhaseID `json:"phase"`
	// EventType is the type of event that occurred
	Event EventType `json:"event"`
	// Status is the status to which the transaction changed
	Status TransactionStatus `json:"status"`
}

// TransactionPhaseID is a transaction phase identifier
type TransactionPhaseID string

const (
	// TransactionPhaseChange indicates a transaction in the Change phase
	TransactionPhaseChange TransactionPhaseID = "Change"
	// TransactionPhaseRollback indicates a transaction in the Rollback phase
	TransactionPhaseRollback TransactionPhaseID = "Rollback"
)

// TransactionStatus indicates the status of a transaction within a phase
type TransactionStatus string

// UnmarshalJSON  implements the Unmarshaler interface
func (s *TransactionStatus) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != nilString {
		*s = TransactionStatus(value)
	}
	return nil
}

const (
	// TransactionPending indicates a transaction phase is pending
	TransactionPending TransactionStatus = "Pending"
	// TransactionInProgress indicates a transaction phase is in progress
	TransactionInProgress TransactionStatus = "InProgress"
	// TransactionComplete indicates a transaction phase is complete
	TransactionComplete TransactionStatus = "Complete"
	// TransactionAborted indicates a transaction phase has been aborted
	TransactionAborted TransactionStatus = "Aborted"
	// TransactionCanceled indicates a transaction phase has been canceled
	TransactionCanceled TransactionStatus = "Canceled"
	// TransactionFailed indicates a transaction phase failed
	TransactionFailed TransactionStatus = "Failed"
)

// Transactions is a mapping of transaction indexes to transactions
type Transactions map[Index]Transaction

// UnmarshalJSON  implements the Unmarshaler interface
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

// Transaction is the state of a transaction in the transaction log
type Transaction struct {
	// Index is the index of the Transaction within the Transaction log
	Index Index `json:"index"`
	// Phase is the identifier of the current phase of the Transaction
	Phase TransactionPhaseID `json:"phase"`
	// Change is the status of the Transaction's Change phase
	Change TransactionPhase `json:"change"`
	// Rollback is the status of the Transaction's Rollback phase
	Rollback TransactionPhase `json:"rollback"`
}

// TransactionPhase is the state of a transaction for one of the two phases
type TransactionPhase struct {
	Index   Index              `json:"index"`
	Ordinal Ordinal            `json:"ordinal"`
	Values  Values             `json:"values"`
	Commit  *TransactionStatus `json:"commit"`
	Apply   *TransactionStatus `json:"apply"`
}

// Values is a change/rollback values map
type Values map[string]string

// UnmarshalJSON  implements the Unmarshaler interface
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

// Value is a change/rollback value
type Value string

// UnmarshalJSON  implements the Unmarshaler interface
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

// Configuration is the state of a configuration
type Configuration struct {
	State     ConfigurationStatus    `json:"state"`
	Term      Term                   `json:"term"`
	Committed ConfigurationCommitted `json:"committed"`
	Applied   ConfigurationApplied   `json:"applied"`
}

// ConfigurationCommitted is the committed state of a configuration
type ConfigurationCommitted struct {
	Index    Index    `json:"index"`
	Change   Index    `json:"change"`
	Ordinal  Ordinal  `json:"ordinal"`
	Revision Revision `json:"revision"`
	Target   Index    `json:"target"`
	Term     Term     `json:"term"`
	Values   Values   `json:"values"`
}

// ConfigurationApplied is the applied state of a configuration
type ConfigurationApplied struct {
	Index    Index    `json:"index"`
	Ordinal  Ordinal  `json:"ordinal"`
	Revision Revision `json:"revision"`
	Target   Index    `json:"target"`
	Values   Values   `json:"values"`
}

// ConfigurationStatus is the status of a configuration
type ConfigurationStatus string

const (
	// ConfigurationPending indicates a configuration is pending synchronization with the target
	ConfigurationPending ConfigurationStatus = "Pending"
	// ConfigurationComplete indicates a configuration has completed synchronizing with the target
	ConfigurationComplete ConfigurationStatus = "Complete"
)

// Mastership is the state of the mastership election for the model
type Mastership struct {
	Term   Term   `json:"term"`
	Master NodeID `json:"master"`
}
