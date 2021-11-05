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

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	configmodelapi "github.com/onosproject/onos-api/go/onos/configmodel"
	"github.com/onosproject/onos-config/model"
	plugincache "github.com/onosproject/onos-config/model/plugin/cache"
	"github.com/onosproject/onos-config/model/plugin/compiler"
	configmodule "github.com/onosproject/onos-config/model/plugin/module"
	"github.com/onosproject/onos-config/model/registry"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

var log = logging.GetLogger("config-model")

const (
	defaultCachePath    = "/etc/onos/plugins"
	defaultRegistryPath = "/etc/onos/registry"
	defaultModPath      = "/etc/onos/mod"
)

var (
	_, mainFile, _, _ = runtime.Caller(0)
	modRoot           = filepath.Dir(filepath.Dir(filepath.Dir(mainFile)))
)

func main() {
	if err := getCmd().Execute(); err != nil {
		println(err)
		os.Exit(1)
	}
}

func getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "config-model",
	}
	cmd.AddCommand(getRegistryCmd())
	cmd.AddCommand(getInitCmd())
	return cmd
}

func getInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "init",
		Short:        "Initializes the target module info",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			module := configmodule.NewModule(configmodule.Config{Path: modRoot})
			if err := module.Init(); err != nil {
				return err
			}
			return nil
		},
	}
}

func getRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "registry",
	}
	cmd.AddCommand(getRegistryServeCmd())
	cmd.AddCommand(getRegistryGetCmd())
	cmd.AddCommand(getRegistryListCmd())
	cmd.AddCommand(getRegistryPushCmd())
	cmd.AddCommand(getRegistryDeleteCmd())
	return cmd
}

func getRegistryServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Short:        "Start the model registry server",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			caCert, _ := cmd.Flags().GetString("ca-cert")
			cert, _ := cmd.Flags().GetString("cert")
			key, _ := cmd.Flags().GetString("key")
			modPath, _ := cmd.Flags().GetString("mod-path")
			registryPath, _ := cmd.Flags().GetString("registry-path")
			cachePath, _ := cmd.Flags().GetString("cache-path")
			port, _ := cmd.Flags().GetInt16("port")

			server := northbound.NewServer(&northbound.ServerConfig{
				CaPath:      &caCert,
				CertPath:    &cert,
				KeyPath:     &key,
				Port:        port,
				Insecure:    true,
				SecurityCfg: &northbound.SecurityConfig{},
			})

			moduleConfig := configmodule.Config{
				Path: modPath,
			}
			module := configmodule.NewModule(moduleConfig)
			err := module.Init()
			if err != nil {
				return err
			}

			cacheConfig := plugincache.Config{
				Path: cachePath,
			}
			cache, err := plugincache.NewPluginCache(module, cacheConfig)
			if err != nil {
				return err
			}

			compilerConfig := plugincompiler.Config{}
			compiler := plugincompiler.NewPluginCompiler(module, compilerConfig)

			registryConfig := modelregistry.Config{
				Path: registryPath,
			}
			registry := modelregistry.NewConfigModelRegistry(registryConfig)

			service := modelregistry.NewService(registry, cache, compiler)
			server.AddService(service)

			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-c
				os.Exit(0)
			}()

			log.Infof("Starting registry server at '%s'", registryPath)
			err = server.Serve(func(address string) {
				log.Infof("Serving models at '%s' on %s", registryPath, address)
			})
			if err != nil {
				log.Errorf("Registry serve failed: %v", err)
				return err
			}
			return nil
		},
	}
	cmd.Flags().Int16P("port", "p", 5151, "the registry service port")
	cmd.Flags().String("mod-path", defaultModPath, "the path to the onos-config module for which to build plugins")
	cmd.Flags().String("registry-path", defaultRegistryPath, "the path in which to store the registry models")
	cmd.Flags().String("cache-path", defaultCachePath, "the path in which to store the plugins")
	cmd.Flags().String("ca-cert", "", "the CA certificate")
	cmd.Flags().String("cert", "", "the certificate")
	cmd.Flags().String("key", "", "the key")
	return cmd
}

func getRegistryGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "get",
		Short:        "Get a model from the registry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			name, _ := cmd.Flags().GetString("name")
			version, _ := cmd.Flags().GetString("version")

			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()

			client := configmodelapi.NewConfigModelRegistryServiceClient(conn)
			request := &configmodelapi.GetModelRequest{
				Name:    name,
				Version: version,
			}
			ctx, cancel := newContext()
			defer cancel()
			response, err := client.GetModel(ctx, request)
			if err != nil {
				return err
			}

			var moduleInfos []configmodel.ModuleInfo
			for _, module := range response.Model.Modules {
				moduleInfos = append(moduleInfos, configmodel.ModuleInfo{
					Name:         configmodel.Name(module.Name),
					Organization: module.Organization,
					Revision:     configmodel.Revision(module.Revision),
					File:         module.File,
				})
			}

			modelInfo := configmodel.ModelInfo{
				Name:    configmodel.Name(response.Model.Name),
				Version: configmodel.Version(response.Model.Version),
				Modules: moduleInfos,
				Plugin: configmodel.PluginInfo{
					Name:    configmodel.Name(response.Model.Name),
					Version: configmodel.Version(response.Model.Version),
				},
			}

			bytes, err := json.MarshalIndent(modelInfo, "", "  ")
			if err != nil {
				return err
			}
			println(string(bytes))
			return nil
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5151", "the registry address")
	cmd.Flags().StringP("name", "n", "", "the model name")
	cmd.Flags().StringP("version", "v", "", "the model version")
	return cmd
}

func getRegistryListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List models in the registry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := configmodelapi.NewConfigModelRegistryServiceClient(conn)
			request := &configmodelapi.ListModelsRequest{}
			ctx, cancel := newContext()
			defer cancel()
			response, err := client.ListModels(ctx, request)
			if err != nil {
				return err
			}
			for _, modelInfo := range response.Models {
				var moduleInfos []configmodel.ModuleInfo
				for _, module := range modelInfo.Modules {
					moduleInfos = append(moduleInfos, configmodel.ModuleInfo{
						Name:         configmodel.Name(module.Name),
						Organization: module.Organization,
						Revision:     configmodel.Revision(module.Revision),
						File:         module.File,
					})
				}
				model := configmodel.ModelInfo{
					Name:    configmodel.Name(modelInfo.Name),
					Version: configmodel.Version(modelInfo.Version),
					Modules: moduleInfos,
					Plugin: configmodel.PluginInfo{
						Name:    configmodel.Name(modelInfo.Name),
						Version: configmodel.Version(modelInfo.Version),
					},
				}
				bytes, err := json.MarshalIndent(model, "", "  ")
				if err != nil {
					return err
				}
				println(string(bytes))
			}
			return nil
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5151", "the registry address")
	return cmd
}

func getRegistryPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "push",
		Short:        "Push a model to the registry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			name, _ := cmd.Flags().GetString("name")
			version, _ := cmd.Flags().GetString("version")
			files, _ := cmd.Flags().GetStringSlice("file")
			modules, _ := cmd.Flags().GetStringToString("module")
			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := configmodelapi.NewConfigModelRegistryServiceClient(conn)
			model := &configmodelapi.ConfigModel{
				Name:    name,
				Version: version,
				Modules: []*configmodelapi.ConfigModule{},
			}

			for _, path := range files {
				data, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				model.Files[filepath.Base(path)] = string(data)
			}

			for nameRevision, file := range modules {
				names := strings.Split(nameRevision, "@")
				if len(names) != 2 {
					return errors.New("module name must be in the format $name@$revision")
				}
				name, revision := names[0], names[1]
				model.Modules = append(model.Modules, &configmodelapi.ConfigModule{
					Name:     name,
					Revision: revision,
					File:     file,
				})
			}

			request := &configmodelapi.PushModelRequest{
				Model: model,
			}
			ctx, cancel := newContext()
			defer cancel()
			_, err = client.PushModel(ctx, request)
			return err
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5151", "the registry address")
	cmd.Flags().StringP("name", "n", "", "the model name")
	cmd.Flags().StringP("revision", "r", "", "the model revision")
	cmd.Flags().StringSliceP("file", "f", []string{}, "model files")
	cmd.Flags().StringToStringP("module", "m", map[string]string{}, "model module descriptors")
	return cmd
}

func getRegistryDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete",
		Short:        "Delete a model from the registry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			address, _ := cmd.Flags().GetString("address")
			name, _ := cmd.Flags().GetString("name")
			version, _ := cmd.Flags().GetString("version")
			conn, err := connect(address)
			if err != nil {
				return err
			}
			defer conn.Close()
			client := configmodelapi.NewConfigModelRegistryServiceClient(conn)
			request := &configmodelapi.DeleteModelRequest{
				Name:    name,
				Version: version,
			}
			ctx, cancel := newContext()
			defer cancel()
			_, err = client.DeleteModel(ctx, request)
			return err
		},
	}
	cmd.Flags().StringP("address", "a", "localhost:5151", "the registry address")
	cmd.Flags().StringP("name", "n", "", "the model name")
	cmd.Flags().StringP("version", "v", "", "the model version")
	return cmd
}

func connect(address string) (*grpc.ClientConn, error) {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	// Connect to the first matching service
	return grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
}

func newContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()
	return ctx, cancel
}
