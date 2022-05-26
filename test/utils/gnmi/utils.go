// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package gnmi

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"

	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/onosproject/onos-test/pkg/onostest"

	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-api/go/onos/config/v2"
	protoutils "github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
)

const (
	// SimulatorTargetVersion default version for simulated target
	SimulatorTargetVersion = "1.0.0"
	// SimulatorTargetType type for simulated target
	SimulatorTargetType = "devicesim"

	defaultGNMITimeout = time.Second * 30

	// Maximum time for an entire test to complete
	defaultTestTimeout = 3 * time.Minute
)

// MakeContext returns a new context for use in GNMI requests
func MakeContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()
	return context.WithTimeout(ctx, defaultTestTimeout)
}

// NewSimulatorTargetEntity creates a topo entity for a device simulator target
func NewSimulatorTargetEntity(ctx context.Context, simulator *helm.HelmRelease, targetType string, targetVersion string) (*topo.Object, error) {
	simulatorClient := kubernetes.NewForReleaseOrDie(simulator)
	services, err := simulatorClient.CoreV1().Services().List(ctx)
	if err != nil {
		return nil, err
	}
	service := services[0]
	return NewTargetEntity(simulator.Name(), targetType, targetVersion, service.Ports()[0].Address(true))
}

// NewTargetEntity creates a topo entity with the specified target name, type, version and service address
func NewTargetEntity(name string, targetType string, targetVersion string, serviceAddress string) (*topo.Object, error) {
	o := topo.Object{
		ID:   topo.ID(name),
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{
				KindID: topo.ID(targetType),
			},
		},
	}

	if err := o.SetAspect(&topo.TLSOptions{Insecure: true, Plain: true}); err != nil {
		return nil, err
	}

	timeout := defaultGNMITimeout
	if err := o.SetAspect(&topo.Configurable{
		Type:    targetType,
		Address: serviceAddress,
		Version: targetVersion,
		Timeout: &timeout,
	}); err != nil {
		return nil, err
	}

	return &o, nil
}

// AddTargetToTopo adds a new target to topo
func AddTargetToTopo(ctx context.Context, targetEntity *topo.Object) error {
	client, err := NewTopoClient()
	if err != nil {
		return err
	}
	err = client.Create(ctx, targetEntity)
	return err
}

// GetTargetFromTopo retrieves the specified target entity
func GetTargetFromTopo(ctx context.Context, id topo.ID) (*topo.Object, error) {
	client, err := NewTopoClient()
	if err != nil {
		return nil, err
	}
	return client.Get(ctx, id)
}

// UpdateTargetInTopo updates the target
func UpdateTargetInTopo(ctx context.Context, targetEntity *topo.Object) error {
	client, err := NewTopoClient()
	if err != nil {
		return err
	}
	err = client.Update(ctx, targetEntity)
	return err
}

// UpdateTargetTypeVersion updates the target type and version information in the Configurable aspect
func UpdateTargetTypeVersion(ctx context.Context, id topo.ID, targetType configapi.TargetType, targetVersion configapi.TargetVersion) error {
	entity, err := GetTargetFromTopo(ctx, id)
	if err != nil {
		return err
	}
	configurable := topo.Configurable{}
	err = entity.GetAspect(&configurable)
	if err != nil {
		return err
	}
	configurable.Type = string(targetType)
	configurable.Version = string(targetVersion)
	err = entity.SetAspect(&configurable)
	if err != nil {
		return err
	}
	return UpdateTargetInTopo(ctx, entity)
}

func getKindFilter(kind string) *topo.Filters {
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

func getControlRelationFilter() *topo.Filters {
	return getKindFilter(topo.CONTROLS)
}

// WaitForControlRelation waits to create control relation for a given target
func WaitForControlRelation(ctx context.Context, t *testing.T, predicate func(*topo.Relation, topo.Event) bool, timeout time.Duration) bool {
	cl, err := NewTopoClient()
	assert.NoError(t, err)
	stream := make(chan topo.Event)
	err = cl.Watch(ctx, stream, toposdk.WithWatchFilters(getControlRelationFilter()))
	assert.NoError(t, err)
	for event := range stream {
		if predicate(event.Object.GetRelation(), event) {
			return true
		} // Otherwise, loop and wait for the next topo event
	}

	return false
}

// WaitForTargetAvailable waits for a target to become available
func WaitForTargetAvailable(ctx context.Context, t *testing.T, objectID topo.ID, timeout time.Duration) bool {
	return WaitForControlRelation(ctx, t, func(rel *topo.Relation, event topo.Event) bool {
		if rel.TgtEntityID != objectID {
			t.Logf("Topo %v event from %s (expected %s). Discarding\n", event.Type, rel.TgtEntityID, objectID)
			return false
		}

		if event.Type == topo.EventType_ADDED || event.Type == topo.EventType_UPDATED || event.Type == topo.EventType_NONE {
			cl, err := NewTopoClient()
			assert.NoError(t, err)
			_, err = cl.Get(ctx, event.Object.ID)
			if err == nil {
				t.Logf("Target %s is available", objectID)
				return true
			}
		}

		return false
	}, timeout)
}

// WaitForTargetUnavailable waits for a target to become available
func WaitForTargetUnavailable(ctx context.Context, t *testing.T, objectID topo.ID, timeout time.Duration) bool {
	return WaitForControlRelation(ctx, t, func(rel *topo.Relation, event topo.Event) bool {
		if rel.TgtEntityID != objectID {
			t.Logf("Topo %v event from %s (expected %s). Discarding\n", event, rel.TgtEntityID, objectID)
			return false
		}

		if event.Type == topo.EventType_REMOVED || event.Type == topo.EventType_NONE {
			cl, err := NewTopoClient()
			assert.NoError(t, err)
			_, err = cl.Get(ctx, event.Object.ID)
			if errors.IsNotFound(err) {
				t.Logf("Target %s is unavailable", objectID)
				return true
			}
		}
		return false
	}, timeout)
}

// WaitForRollback waits for a COMPLETED status on the most recent rollback transaction
func WaitForRollback(ctx context.Context, t *testing.T, transactionIndex v2.Index, wait time.Duration) bool {
	client, err := NewTransactionServiceClient(ctx)
	assert.NoError(t, err)

	stream, err := client.WatchTransactions(ctx, &admin.WatchTransactionsRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	start := time.Now()

	for {
		resp, err := stream.Recv()
		if err != nil {
			return false
		}
		assert.NotNil(t, resp)
		fmt.Printf("%v\n", resp.TransactionEvent)

		t := resp.TransactionEvent.Transaction
		if rt := t.GetRollback(); rt != nil {
			if rt.RollbackIndex == transactionIndex {
				return true
			}
		}

		if time.Since(start) > wait {
			return false
		}
	}
}

// SyncExtension returns list of extensions with just the transaction mode extension set to sync and atomic.
func SyncExtension(t *testing.T) []*gnmi_ext.Extension {
	return []*gnmi_ext.Extension{TransactionStrategyExtension(t, configapi.TransactionStrategy_SYNCHRONOUS, 0)}
}

// TransactionStrategyExtension returns a transaction strategy extension populated with the specified fields
func TransactionStrategyExtension(t *testing.T,
	synchronicity configapi.TransactionStrategy_Synchronicity,
	isolation configapi.TransactionStrategy_Isolation) *gnmi_ext.Extension {
	ext := v2.TransactionStrategy{
		Synchronicity: synchronicity,
		Isolation:     isolation,
	}
	b, err := ext.Marshal()
	assert.NoError(t, err)
	return &gnmi_ext.Extension{
		Ext: &gnmi_ext.Extension_RegisteredExt{
			RegisteredExt: &gnmi_ext.RegisteredExtension{
				Id:  v2.TransactionStrategyExtensionID,
				Msg: b,
			},
		},
	}
}

// GetTargetPath creates a target path
func GetTargetPath(target string, path string) []protoutils.TargetPath {
	return GetTargetPathWithValue(target, path, "", "")
}

// GetTargetPathWithValue creates a target path with a value to set
func GetTargetPathWithValue(target string, path string, value string, valueType string) []protoutils.TargetPath {
	targetPath := make([]protoutils.TargetPath, 1)
	targetPath[0].TargetName = target
	targetPath[0].Path = path
	targetPath[0].PathDataValue = value
	targetPath[0].PathDataType = valueType
	return targetPath
}

// GetTargetPaths creates multiple target paths
func GetTargetPaths(targets []string, paths []string) []protoutils.TargetPath {
	var targetPaths = make([]protoutils.TargetPath, len(paths)*len(targets))
	pathIndex := 0
	for _, dev := range targets {
		for _, path := range paths {
			targetPaths[pathIndex].TargetName = dev
			targetPaths[pathIndex].Path = path
			pathIndex++
		}
	}
	return targetPaths
}

// GetTargetPathsWithValues creates multiple target paths with values to set
func GetTargetPathsWithValues(targets []string, paths []string, values []string) []protoutils.TargetPath {
	var targetPaths = GetTargetPaths(targets, paths)
	valueIndex := 0
	for range targets {
		for _, value := range values {
			targetPaths[valueIndex].PathDataValue = value
			targetPaths[valueIndex].PathDataType = protoutils.StringVal
			valueIndex++
		}
	}
	return targetPaths
}

// MakeProtoPath returns a Path: element for a given target and Path
func MakeProtoPath(target string, path string) string {
	var protoBuilder strings.Builder

	protoBuilder.WriteString("path: ")
	gnmiPath := protoutils.MakeProtoTarget(target, path)
	protoBuilder.WriteString(gnmiPath)
	return protoBuilder.String()
}

// CreateSimulator creates a device simulator
func CreateSimulator(ctx context.Context, t *testing.T) *helm.HelmRelease {
	return CreateSimulatorWithName(ctx, t, random.NewPetName(2), true)
}

// CreateSimulatorWithName creates a device simulator
func CreateSimulatorWithName(ctx context.Context, t *testing.T, name string, createTopoEntity bool) *helm.HelmRelease {
	simulator := helm.
		Chart("device-simulator", onostest.OnosChartRepo).
		Release(name).
		Set("image.tag", "latest")
	err := simulator.Install(true)
	assert.NoError(t, err, "could not install device simulator %v", err)

	time.Sleep(2 * time.Second)

	if createTopoEntity {
		simulatorTarget, err := NewSimulatorTargetEntity(ctx, simulator, SimulatorTargetType, SimulatorTargetVersion)
		assert.NoError(t, err, "could not make target for simulator %v", err)

		err = AddTargetToTopo(ctx, simulatorTarget)
		assert.NoError(t, err, "could not add target to topo for simulator %v", err)
	}

	return simulator
}

// DeleteSimulator shuts down the simulator pod and removes the target from topology
func DeleteSimulator(t *testing.T, simulator *helm.HelmRelease) {
	assert.NoError(t, simulator.Uninstall())
}
