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
	atomix := helm.
		Chart("atomix-controller").
		Release("atomix-controller").
		Set("namespace", helm.Namespace())
	err := atomix.Install(true)
	if err != nil {
		return err
	}

	controller := "atomix-controller:5679"

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
	client, err := getGNMIClient()
	if err != nil {
		return err
	}
	s.client = client
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

const clientCert = `
-----BEGIN CERTIFICATE-----
MIIDZTCCAk0CCQDl7NF6ekffcTANBgkqhkiG9w0BAQsFADByMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCU1lbmxvUGFyazEMMAoGA1UECgwDT05G
MRQwEgYDVQQLDAtFbmdpbmVlcmluZzEeMBwGA1UEAwwVY2Eub3Blbm5ldHdvcmtp
bmcub3JnMB4XDTE5MDQxMTExMTYyM1oXDTIwMDQxMDExMTYyM1owdzELMAkGA1UE
BhMCVVMxCzAJBgNVBAgMAkNBMRIwEAYDVQQHDAlNZW5sb1BhcmsxDDAKBgNVBAoM
A09ORjEUMBIGA1UECwwLRW5naW5lZXJpbmcxIzAhBgNVBAMMGmNsaWVudDEub3Bl
bm5ldHdvcmtpbmcub3JnMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
5mR12oGXP+uDD7DzQZdTg96eHWTc0UKPwie2I5LLLVsRoH2PO5s2B5r6r/E8OUG4
0pGb6tkDRIJ8eC0Z/6NvBkzn4fsJ5g0UW6sVlXfaf0y9JnMSvV05+g++75a7+CRx
1BG3GNjGWbke1mx8d6SrQ8D1sjI3L0D+32mi0WU9jO2Uw9YXvXgxQmL9Krxdr3M/
aZO9sTJZtIT0EEY3qBpPv+daAbuP5m+uhiEzYZP2bLywyzGyfrUmj9fjG/D1kuMM
haEIUJQ2VTcIApKG/Kb3Mk3b3VCfTvpEHMVrKMoyNHQXXi+6X106+cu2WtoPv+U5
VFVoufjRWSbcOmQ7qIHBiwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBRBR6LTFEU
SWeEeguMsbHxN/6NIZuPejib1q9fTHeZ9cnIHIOLJaZzHiMZn5uw8s6D26kveNps
iCr4O8xOjUa0uwbhMTgm3wkODLlV1DwGjFWk8v5UKGWqUQ94wVMQ16YMIR5DgJJM
0DUzVcoFz+vLnMrDZ0AEk5vra1Z5KweSRvwHX7dJ6FIW7X3IgqXTqJtlV/D/vIi3
UfBnjzqOy2LVfBD7du7i5NbTHfTUpoTvddVwQaKCuQGYHocoQvQD3VQcQDh1u0DD
n2GkeEDLaDAGFAIO+PDg2iT8BhKeEepqswid9gYAhZcOjrlnl6smZo7jEzBj1a9Q
e3q1STjfQqe8
-----END CERTIFICATE-----
`

const clientKey = `
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDmZHXagZc/64MP
sPNBl1OD3p4dZNzRQo/CJ7YjksstWxGgfY87mzYHmvqv8Tw5QbjSkZvq2QNEgnx4
LRn/o28GTOfh+wnmDRRbqxWVd9p/TL0mcxK9XTn6D77vlrv4JHHUEbcY2MZZuR7W
bHx3pKtDwPWyMjcvQP7faaLRZT2M7ZTD1he9eDFCYv0qvF2vcz9pk72xMlm0hPQQ
RjeoGk+/51oBu4/mb66GITNhk/ZsvLDLMbJ+tSaP1+Mb8PWS4wyFoQhQlDZVNwgC
kob8pvcyTdvdUJ9O+kQcxWsoyjI0dBdeL7pfXTr5y7Za2g+/5TlUVWi5+NFZJtw6
ZDuogcGLAgMBAAECggEBAIc9VUjsZSJqVsaxMjnAYI+578qFWHGlxslLkkkTdByt
po005w0wMOkJ+jmpO5bIk3tXadTTim1+wx2wK+C5yQRDxKIMQGVALEEbDlJsxl+P
ZkDZr5hkzxGQiJ4PN0uT6RV5SKdXKCem2Qk5KV751GazMAZoH6inWHVAhwiviw/b
kSJmXcQifxB9R5Br+yCdkRNGg+EtadxAkRtZdW0N0H6LwWxsl32I4o1WM3N2Tyag
kpKPPZ5J5U+279Rpz7W4JAbGzWBOL0Wc2pz5p+aKVTWia0MoqzHR4P4YnkGM+w9Y
j6+Nemdedx62KPhOnQH1uvuG3vnOtt2Ss5OLxePgmjECgYEA9bVguF1D5rpp6MSK
2izZt0mNqhiozm84W2UrAwDhtW5tptW2JBPj2T05+PbEOUEgsvucWfmhZoBXNRCw
IlLQZh46LJFXyW1Awn3PuYquruF61phDoqU9Ou5skJrh0ez+vX872HkH4KW3MfWq
w3LW4qXt6z+lBgPY8hNAlis3WE0CgYEA8Ara5J915ZoVll1As84H61NHmkyMFENh
PjUJqL6tPxvZ+lkBeA157o6mrIgNmG5bLnzonpT4rqemewxEYL39sJ6CVzHRFy8I
F0VNLzZbYizrPLRvT+Gkh0jf6W7Iarzmcdb8cMDxQ+9LmwR/Q3XAD8ntqzrbwVl5
FOZlGq2ZbTcCgYEAuMULlbi07hXyvNLH4+dkVXufZ3EhyBNFGx2J6blJAkmndZUy
YhD+/4cWSE0xJCkAsPebDOI26EDM05/YBAeopZJHhupJTLS2xUsc4VcTo3j2Cdf4
zJ9b2yweQePmuxlwOwop89CYBuw3Rf+KyW1bgJbswkJbE5njE688m3CmLuUCgYAf
K2mtEj++5rky4z0JnBFPL2s20AXIg89WwpBUhx37+ePeLDySmD1jCsb91FTfnETe
zn1uSi3YkBCAHeGrJkCQ9KQ8Kk3aUtMcInWZUdef8fFB2rQxjT1OC9p3d1ky8wCB
e8cf5Q3vIl2Q7Y6Q9fNQmYnxGB19B98/JYOvaSdpFQKBgFBJ+tdJ5ghXSdvAzGno
trQlL1AYW/kYsxZaALd1R+vK3vxeHOtUWiq3923QttYsVXPRQe1TEEdxlOb7+hwE
g5NVOIsDpB1OqjQRb9PjipANkHQRKgrYFB20ZQUoaOMckhlVyqE6WcanGpUxJ0xg
1F0itWrqPGEs83BRQI/aLlsj
-----END PRIVATE KEY-----
`

// getClientCredentials returns the credentials for a service client
func getClientCredentials() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}
