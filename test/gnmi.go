// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

// WaitForControlRelation waits to create control relation for a given target
func (s *Suite) WaitForControlRelation(predicate func(*topo.Relation, topo.Event) bool) bool {
	cl, err := s.NewTopoClient()
	s.NoError(err)
	stream := make(chan topo.Event)
	err = cl.Watch(s.Context(), stream, toposdk.WithWatchFilters(s.getControlRelationFilter()))
	s.NoError(err)
	for event := range stream {
		if predicate(event.Object.GetRelation(), event) {
			return true
		} // Otherwise, loop and wait for the next topo event
	}
	return false
}

func (s *Suite) getKindFilter(kind string) *topo.Filters {
	kindFilter := &topo.Filters{
		KindFilter: &topo.Filter{
			Filter: &topo.Filter_Equal_{
				Equal_: &topo.EqualFilter{
					Value: kind,
				},
			},
		},
	}
	return kindFilter
}

func (s *Suite) getControlRelationFilter() *topo.Filters {
	return s.getKindFilter(topo.CONTROLS)
}

// WaitForTargetAvailable waits for a target to become available
func (s *Suite) WaitForTargetAvailable(objectID topo.ID) bool {
	return s.WaitForControlRelation(func(rel *topo.Relation, event topo.Event) bool {
		if rel.TgtEntityID != objectID {
			s.T().Logf("Topo %v event from %s (expected %s). Discarding\n", event.Type, rel.TgtEntityID, objectID)
			return false
		}

		if event.Type == topo.EventType_ADDED || event.Type == topo.EventType_UPDATED || event.Type == topo.EventType_NONE {
			cl, err := s.NewTopoClient()
			s.NoError(err)
			_, err = cl.Get(s.Context(), event.Object.ID)
			if err == nil {
				s.T().Logf("Target %s is available", objectID)
				return true
			}
		}

		return false
	})
}

// WaitForTargetUnavailable waits for a target to become available
func (s *Suite) WaitForTargetUnavailable(objectID topo.ID) bool {
	return s.WaitForControlRelation(func(rel *topo.Relation, event topo.Event) bool {
		if rel.TgtEntityID != objectID {
			s.T().Logf("Topo %v event from %s (expected %s). Discarding\n", event, rel.TgtEntityID, objectID)
			return false
		}

		if event.Type == topo.EventType_REMOVED || event.Type == topo.EventType_NONE {
			cl, err := s.NewTopoClient()
			s.NoError(err)
			_, err = cl.Get(s.Context(), event.Object.ID)
			if errors.IsNotFound(err) {
				s.T().Logf("Target %s is unavailable", objectID)
				return true
			}
		}
		return false
	})
}

// WaitForRollback waits for a COMPLETED status on the most recent rollback transaction
func (s *Suite) WaitForRollback(transactionIndex configapi.Index) bool {
	client, err := s.NewTransactionServiceClient()
	s.NoError(err)

	stream, err := client.WatchTransactions(s.Context(), &admin.WatchTransactionsRequest{})
	s.NoError(err)
	s.NotNil(stream)

	for {
		resp, err := stream.Recv()
		if err != nil {
			return false
		}
		s.NotNil(resp)
		fmt.Printf("%v\n", resp.TransactionEvent)

		t := resp.TransactionEvent.Transaction
		if rt := t.GetRollback(); rt != nil {
			if rt.RollbackIndex == transactionIndex {
				return true
			}
		}
	}
}

// SyncExtension returns list of extensions with just the transaction mode extension set to sync and atomic.
func (s *Suite) SyncExtension() []*gnmi_ext.Extension {
	return []*gnmi_ext.Extension{s.TransactionStrategyExtension(configapi.TransactionStrategy_SYNCHRONOUS, 0)}
}

// TransactionStrategyExtension returns a transaction strategy extension populated with the specified fields
func (s *Suite) TransactionStrategyExtension(
	synchronicity configapi.TransactionStrategy_Synchronicity,
	isolation configapi.TransactionStrategy_Isolation) *gnmi_ext.Extension {
	ext := configapi.TransactionStrategy{
		Synchronicity: synchronicity,
		Isolation:     isolation,
	}
	b, err := ext.Marshal()
	s.NoError(err)
	return &gnmi_ext.Extension{
		Ext: &gnmi_ext.Extension_RegisteredExt{
			RegisteredExt: &gnmi_ext.RegisteredExtension{
				Id:  configapi.TransactionStrategyExtensionID,
				Msg: b,
			},
		},
	}
}
