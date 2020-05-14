// Copyright 2019-present Open Networking Foundation.
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

package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/onosproject/onos-config/pkg/config/load"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/openconfig/gnmi/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)

const (
	addressFlag     = "service-address"
	addressKey      = "service-address"
	tlsCertPathFlag = "tls-cert-path"
	tlsKeyPathFlag  = "tls-key-path"
	noTLSFlag       = "no-tls"
)

func getLoadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load {type}",
		Short: "Load configuration from a file",
	}
	cmd.AddCommand(getYamlCommand())
	cmd.AddCommand(getProtoCommand())
	return cmd
}

func getYamlCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "yaml {file(s)} [args]",
		Short: "Load configuration from one or more YAML files",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runLoadYamlCommand,
	}
	return cmd
}

func runLoadYamlCommand(cmd *cobra.Command, args []string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	dest := &client.Destination{
		Addrs: []string{getAddress(cmd)},
	}
	if !noTLS(cmd) {
		dest.TLS = &tls.Config{
			InsecureSkipVerify: true,
		}
		cert := getCertPath(cmd)
		key := getKeyPath(cmd)
		dest.TLS.RootCAs, _ = certs.GetCertPoolDefault()
		var clientCerts tls.Certificate
		var err error
		if cert == "" && key == "" {
			clientCerts, _ = tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
		} else {
			clientCerts, err = tls.LoadX509KeyPair(cert, key)
			if err != nil {
				return err
			}
		}
		dest.TLS.Certificates = []tls.Certificate{clientCerts}
	}

	gnmiClient, err := southbound.GnmiClientFactory(ctx, *dest)
	if err != nil {
		fmt.Printf("Error loading gnmiClient at %v\n", dest.Addrs)
		return err
	}

	for _, arg := range args {
		fmt.Printf("Loading config %s\n", arg)
		configGnmi, err := load.GetConfigGnmi(arg)
		if err != nil {
			fmt.Printf("Error loading %s\n", arg)
			return err
		}
		gnmiSetRequest := load.ToGnmiSetRequest(&configGnmi)
		resp, err := gnmiClient.Set(ctx, gnmiSetRequest)
		if err != nil {
			fmt.Printf("Error running set on %s \n%v\n", arg, configGnmi.SetRequest)
			return err
		}
		if len(resp.Response) == 0 {
			return fmt.Errorf("empty response to gNMI Set %v", resp)
		}
		fmt.Printf("Load succeeded %v\n", resp.GetExtension())
	}

	return nil
}

func getProtoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proto {file(s)} [args]",
		Short: "Load configuration from one or more gNMI proto files",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runLoadProtoCommand,
	}
	return cmd
}

func runLoadProtoCommand(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("not yet implemented")
}

func getAddress(cmd *cobra.Command) string {
	address, _ := cmd.Flags().GetString(addressFlag)
	if address == "" {
		return viper.GetString(addressKey)
	}
	return address
}

func getCertPath(cmd *cobra.Command) string {
	certPath, _ := cmd.Flags().GetString(tlsCertPathFlag)
	return certPath
}

func getKeyPath(cmd *cobra.Command) string {
	keyPath, _ := cmd.Flags().GetString(tlsKeyPathFlag)
	return keyPath
}

func noTLS(cmd *cobra.Command) bool {
	tls, _ := cmd.Flags().GetBool(noTLSFlag)
	return tls
}
