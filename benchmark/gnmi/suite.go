// Copyright 2020-present Open Networking Foundation.
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

package gnmi

import (
	"context"
	"crypto/tls"
	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	gclient "github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/client/gnmi"
	"time"
)

// BenchmarkSuite is an onos-config gNMI benchmark suite
type BenchmarkSuite struct {
	benchmark.Suite
	simulator *helm.HelmRelease
	client    *client.Client
	value     input.Source
}

// SetupSuite :: benchmark
func (s *BenchmarkSuite) SetupSuite(c *benchmark.Context) error {
	// Setup the Atomix controller
	err := helm.Chart("kubernetes-controller", "https://charts.atomix.io").
		Release("onos-config-atomix").
		Set("scope", "Namespace").
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart("raft-storage-controller", "https://charts.atomix.io").
		Release("onos-config-raft").
		Set("scope", "Namespace").
		Install(true)
	if err != nil {
		return err
	}

	controller := "onos-config-atomix-kubernetes-controller:5679"

	// Install the onos-topo chart
	err = helm.
		Chart("onos-topo").
		Release("onos-topo").
		Set("replicaCount", 2).
		Set("store.controller", controller).
		Install(false)
	if err != nil {
		return err
	}

	// Install the onos-config chart
	err = helm.
		Chart("onos-config").
		Release("onos-config").
		Set("replicaCount", 2).
		Set("store.controller", controller).
		Install(true)
	if err != nil {
		return err
	}
	return nil
}

// SetupWorker :: benchmark
func (s *BenchmarkSuite) SetupWorker(c *benchmark.Context) error {
	s.value = input.RandomString(8)
	s.simulator = helm.
		Chart("device-simulator").
		Release(random.NewPetName(2))
	if err := s.simulator.Install(true); err != nil {
		return err
	}
	gnmiClient, err := getGNMIClient()
	if err != nil {
		return err
	}
	s.client = gnmiClient
	return nil
}

// TearDownWorker :: benchmark
func (s *BenchmarkSuite) TearDownWorker(c *benchmark.Context) error {
	s.client.Close()
	return s.simulator.Uninstall()
}

var _ benchmark.SetupWorker = &BenchmarkSuite{}

func getDestination() (gclient.Destination, error) {
	creds, err := getClientCredentials()
	if err != nil {
		return gclient.Destination{}, err
	}
	onosConfig := helm.Release("onos-config")
	onosConfigClient := kubernetes.NewForReleaseOrDie(onosConfig)
	services, err := onosConfigClient.CoreV1().Services().List()
	if err != nil {
		return gclient.Destination{}, err
	}
	service := services[0]
	return gclient.Destination{
		Addrs:   []string{service.Ports()[0].Address(true)},
		Target:  service.Name,
		TLS:     creds,
		Timeout: 10 * time.Second,
	}, nil
}

// getGNMIClient makes a GNMI client to use for requests
func getGNMIClient() (*client.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dest, err := getDestination()
	if err != nil {
		return nil, err
	}
	gnmiClient, err := client.New(ctx, dest)
	if err != nil {
		return nil, err
	}
	return gnmiClient.(*client.Client), nil
}

// getClientCredentials returns the credentials for a service client
func getClientCredentials() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}
