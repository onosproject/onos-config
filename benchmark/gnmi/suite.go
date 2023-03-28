// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-config/benchmark/utils/gnmi"
	"github.com/onosproject/onos-test/pkg/onostest"
	"github.com/openconfig/gnmi/client/gnmi"
	"time"
)

// BenchmarkSuite is an onos-config gNMI benchmark suite
type BenchmarkSuite struct {
	benchmark.Suite
	umbrella  *helm.Release
	simulator *helm.Release
	client    *client.Client
}

// SetupSuite :: benchmark
func (s *BenchmarkSuite) SetupSuite(ctx context.Context) error {
	release, err := s.Helm().Install("onos", "onos-umbrella").
		RepoURL(onostest.OnosChartRepo).
		Set("onos-topo.image.tag", "latest").
		Set("onos-config.image.tag", "latest").
		Set("onos-config-model.image.tag", "latest").
		Set("onos-topo.replicaCount", 2).
		Set("onos-config.replicaCount", 2).
		Wait().
		Get(ctx)
	if err != nil {
		return err
	}
	s.umbrella = release
	return nil
}

// TearDownSuite :: benchmark
func (s *BenchmarkSuite) TearDownSuite(ctx context.Context) error {
	return s.Helm().Uninstall(s.umbrella.Name).Do(ctx)
}

// SetupWorker :: benchmark
func (s *BenchmarkSuite) SetupWorker(ctx context.Context) error {
	release, err := s.Helm().
		Install(petname.Generate(2, "-"), "device-simulator").
		RepoURL(onostest.OnosChartRepo).
		Wait().
		Get(ctx)
	if err != nil {
		return err
	}
	s.simulator = release
	gnmiClient, err := getGNMIClient()
	if err != nil {
		return err
	}
	s.client = gnmiClient
	return nil
}

// TearDownWorker :: benchmark
func (s *BenchmarkSuite) TearDownWorker(ctx context.Context) error {
	s.client.Close()
	return s.Helm().Uninstall(s.simulator.Name).Do(ctx)
}

var _ benchmark.SetupWorker = &BenchmarkSuite{}

// getGNMIClient makes a GNMI client to use for requests
func getGNMIClient() (*client.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dest, err := gnmi.GetOnosConfigDestination()
	if err != nil {
		return nil, err
	}
	gnmiClient, err := client.New(ctx, dest)
	if err != nil {
		return nil, err
	}
	return gnmiClient.(*client.Client), nil
}

var _ benchmark.SetupSuite = (*BenchmarkSuite)(nil)
var _ benchmark.TearDownSuite = (*BenchmarkSuite)(nil)
var _ benchmark.SetupWorker = (*BenchmarkSuite)(nil)
var _ benchmark.TearDownWorker = (*BenchmarkSuite)(nil)
