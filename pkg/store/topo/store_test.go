// Copyright 2021-present Open Networking Foundation.
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

package topo

import (
	"context"
	"fmt"
	atomix "github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	topomgr "github.com/onosproject/onos-topo/pkg/manager"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

const (
	caCrtFile = "/tmp/onos-proxy.cacrt"
	crtFile   = "/tmp/onos-proxy.crt"
	keyFile   = "/tmp/onos-proxy.key"
)

func writeFile(file string, s string) {
	err := ioutil.WriteFile(file, []byte(s), 0644)
	if err != nil {
		fmt.Printf("error writing generated code to file: %s\n", err)
		os.Exit(-1)
	}
}

func createTopoServer(t *testing.T) (Store, *topomgr.Manager, *atomix.Test, error) {
	test := atomix.NewTest(rsm.NewProtocol(), atomix.WithReplicas(1), atomix.WithPartitions(1))
	assert.NoError(t, test.Start())

	atomixClient, err := test.NewClient("test")
	assert.NoError(t, err)

	writeFile(caCrtFile, certs.OnfCaCrt)
	writeFile(crtFile, certs.DefaultOnosConfigCrt)
	writeFile(keyFile, certs.DefaultOnosConfigKey)

	config := topomgr.Config{CAPath: caCrtFile, CertPath: crtFile, KeyPath: keyFile, GRPCPort: 5150, AtomixClient: atomixClient}
	mgr := topomgr.NewManager(config)
	go mgr.Run()

	opts, err := certs.HandleCertPaths(config.CAPath, config.KeyPath, config.CertPath, true)
	assert.NoError(t, err)

	store, err := NewStore("localhost:5150", opts...)
	assert.NoError(t, err)
	assert.NotNil(t, store)

	return store, mgr, test, err
}

func Test_Basics(t *testing.T) {
	store, mgr, test, err := createTopoServer(t)
	defer test.Stop()
	defer mgr.Close()

	o1 := &topoapi.Object{
		ID: "foo", Type: topoapi.Object_ENTITY, Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{
				KindID: "test",
			},
		},
	}

	o2 := &topoapi.Object{
		ID: "bar", Type: topoapi.Object_ENTITY, Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{
				KindID: "test",
			},
		},
	}

	// Add one object to start
	err = store.Create(context.TODO(), o1)
	assert.NoError(t, err)

	// Setup a watch for topo events
	ch := make(chan topoapi.Event)
	err = store.Watch(context.TODO(), ch, &topoapi.Filters{})
	assert.NoError(t, err)

	// Check that we got the replay event
	e := <-ch
	assert.Equal(t, e.Type, topoapi.EventType_NONE)
	assert.Equal(t, e.Object.ID, topoapi.ID("foo"))

	// Create another object
	err = store.Create(context.TODO(), o2)
	assert.NoError(t, err)

	// Check that we got an added event
	e = <-ch
	assert.Equal(t, e.Type, topoapi.EventType_ADDED)
	assert.Equal(t, e.Object.ID, topoapi.ID("bar"))

	// List objects and make sure we got two
	objects, err := store.List(context.TODO(), &topoapi.Filters{})
	assert.NoError(t, err)
	assert.NotNil(t, objects)
	assert.Equal(t, len(objects), 2)

	// Get an object
	o3, err := store.Get(context.TODO(), topoapi.ID("bar"))

	// Update an object with a new aspect
	err = o3.SetAspect(&topoapi.Configurable{
		Type:    "app",
		Address: "nowhere.local",
		Target:  "bar",
		Version: "0.1",
		Timeout: 20,
	})
	assert.NoError(t, err)
	err = store.Update(context.TODO(), o3)
	assert.NoError(t, err)

	// Check that we got an update event
	e = <-ch
	assert.Equal(t, e.Type, topoapi.EventType_UPDATED)
	assert.Equal(t, e.Object.ID, topoapi.ID("bar"))

	// Remove an object
	err = store.Delete(context.TODO(), o2)
	assert.NoError(t, err)

	// Check that we got a delete event
	e = <-ch
	assert.Equal(t, e.Type, topoapi.EventType_REMOVED)
	assert.Equal(t, e.Object.ID, topoapi.ID("bar"))
}
