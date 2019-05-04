// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

package southbound

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"time"

	"github.com/openconfig/gnmi/client"
	"github.com/opennetworkinglab/onos-config/southbound/topocache"
)

func createDestination(device topocache.Device) (*client.Destination, Key) {
	d := &client.Destination{TLS: &tls.Config{}}
	d.Addrs = []string{device.Addr}
	d.Target = device.Target
	d.Timeout = time.Duration(device.Timeout * time.Second)
	if device.CaPath == "" {
		log.Println("Loading default CA onfca")
		d.TLS.RootCAs = getCertPoolDefault()
	} else {
		d.TLS.RootCAs = getCertPool(device.CaPath)
	}

	if device.CertPath == "" && device.KeyPath == "" {
		// Load default Certificates
		log.Println("Loading default certificates")
		clientCerts, err := tls.X509KeyPair([]byte(defaultClientCrt), []byte(defaultClientKey))
		if err != nil {
			log.Println("Error loading default certs")
		}

		d.TLS.Certificates = []tls.Certificate{clientCerts}
	} else if device.CertPath != "" && device.KeyPath != "" {
		// Load certs given for device
		d.TLS.Certificates = []tls.Certificate{setCertificate(device.CertPath, device.KeyPath)}

	} else if device.Usr != "" && device.Pwd != "" {
		//TODO implement
		// cred := &client.Credentials{}
		// cred.Username = "test"
		// cred.Password = "testpwd"
		//d.Credentials = cred
		//log.Println(*cred)
	} else {
		d.TLS = &tls.Config{InsecureSkipVerify: true}
	}
	return d, Key{Key: device.Addr}
}

func certIDCheck(device topocache.Device) {
	if device.CertID == "" {
		log.Fatal("Must set a certificate ID")
	}
}

func setCertificate(pathCert string, pathKey string) tls.Certificate {
	certificate, err := tls.LoadX509KeyPair(pathCert, pathKey)
	if err != nil {
		log.Println("could not load client key pair", err)
	}
	return certificate
}

func getCertPool(CaPath string) *x509.CertPool {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(CaPath)
	if err != nil {
		log.Println("could not read", CaPath, err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Println("failed to append CA certificates")
	}
	return certPool
}

func getCertPoolDefault() *x509.CertPool {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(onfCaCrt)); !ok {
		log.Println("failed to append CA certificates")
	}
	return certPool
}
