// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	"github.com/cenkalti/backoff"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-api/go/onos/topo"
)

// NewSimulatorTargetEntity creates a topo entity for a device simulator target
func (s *Suite) NewSimulatorTargetEntity(name string, targetType string, targetVersion string) (*topo.Object, error) {
	return s.NewTargetEntity(name, targetType, targetVersion, fmt.Sprintf("%s-device-simulator:11161", name))
}

// NewTargetEntity creates a topo entity with the specified target name, type, version and service address
func (s *Suite) NewTargetEntity(name string, targetType string, targetVersion string, serviceAddress string) (*topo.Object, error) {
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
		Type:                 targetType,
		Address:              serviceAddress,
		Version:              targetVersion,
		Timeout:              &timeout,
		ValidateCapabilities: true,
	}); err != nil {
		return nil, err
	}

	return &o, nil
}

// AddTargetToTopo adds a new target to topo
func (s *Suite) AddTargetToTopo(targetEntity *topo.Object) error {
	client, err := s.NewTopoClient()
	if err != nil {
		return err
	}
	err = client.Create(s.Context(), targetEntity)
	return err
}

// GetTargetFromTopo retrieves the specified target entity
func (s *Suite) GetTargetFromTopo(id topo.ID) (*topo.Object, error) {
	client, err := s.NewTopoClient()
	if err != nil {
		return nil, err
	}
	return client.Get(s.Context(), id)
}

// UpdateTargetInTopo updates the target
func (s *Suite) UpdateTargetInTopo(targetEntity *topo.Object) error {
	client, err := s.NewTopoClient()
	if err != nil {
		return err
	}
	err = client.Update(s.Context(), targetEntity)
	return err
}

// UpdateTargetTypeVersion updates the target type and version information in the Configurable aspect
func (s *Suite) UpdateTargetTypeVersion(id topo.ID, targetType configapi.TargetType, targetVersion configapi.TargetVersion) error {
	updateTargetTypeVersion := func() error {
		entity, err := s.GetTargetFromTopo(id)
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
		return s.UpdateTargetInTopo(entity)
	}

	err := backoff.Retry(updateTargetTypeVersion, backoff.NewExponentialBackOff())
	if err != nil {
		return err
	}
	return nil
}
