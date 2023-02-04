// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/stretchr/testify/assert"
)

const (
	caCrtFile = "/tmp/onos-topo.cacrt"
	crtFile   = "/tmp/onos-topo.crt"
	keyFile   = "/tmp/onos-topo.key"
)

func writeFile(file string, s string) {
	err := ioutil.WriteFile(file, []byte(s), 0644)
	if err != nil {
		fmt.Printf("error writing generated code to file: %s\n", err)
		os.Exit(-1)
	}
}

var store Store

// TODO we need to mock the server rather than depend on onos-topo. It was introducing some dependency issue
/*func TestMain(m *testing.M) {
	atomix := test.NewClient()
	defer atomix.Close()

	writeFile(caCrtFile, certs.OnfCaCrt)
	writeFile(crtFile, certs.DefaultOnosConfigCrt)
	writeFile(keyFile, certs.DefaultOnosConfigKey)

	config := topomgr.Config{
		ServiceFlags: &cli.ServiceEndpointFlags{
			CAPath:   caCrtFile,
			CertPath: crtFile,
			KeyPath:  keyFile,
			BindPort: 5150,
			NoTLS:    true,
		},
	}
	mgr := topomgr.NewManager(config)
	err := mgr.Start()
	if err != nil {
		log.Warn(err)
		os.Exit(1)
	}

	opts, err := certs.HandleCertPaths(config.ServiceFlags.CAPath, config.ServiceFlags.KeyPath, config.ServiceFlags.CertPath, true)
	if err != nil {
		os.Exit(1)
	}

	store, err = NewStore("localhost:5150", opts...)
	if err != nil {
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}*/

func Test_Basics(t *testing.T) {
	t.Skip()
	// Add one object to start
	o1 := &topoapi.Object{
		ID: "foo", Type: topoapi.Object_ENTITY, Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{
				KindID: "test",
			},
		},
	}
	err := store.Create(context.TODO(), o1)
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
	o2 := &topoapi.Object{
		ID: "bar", Type: topoapi.Object_ENTITY, Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{
				KindID: "test",
			},
		},
	}
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
	assert.NoError(t, err)
	assert.Equal(t, o3.ID, topoapi.ID("bar"))

	timeout := time.Second * 20
	// Update an object with a new aspect
	err = o3.SetAspect(&topoapi.Configurable{
		Type:    "app",
		Address: "nowhere.local",
		Target:  "bar",
		Version: "0.1",
		Timeout: &timeout,
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

func Test_BadAdd(t *testing.T) {
	t.Skip()
	o := &topoapi.Object{ID: "bad", Type: topoapi.Object_ENTITY, Obj: &topoapi.Object_Entity{Entity: &topoapi.Entity{}}}
	err := store.Create(context.TODO(), o)
	assert.NoError(t, err)
	err = store.Create(context.TODO(), o)
	assert.Error(t, err)
}

func Test_BadUpdate(t *testing.T) {
	t.Skip()
	o := &topoapi.Object{ID: "nothing", Type: topoapi.Object_ENTITY, Obj: &topoapi.Object_Entity{Entity: &topoapi.Entity{}}}
	err := store.Update(context.TODO(), o)
	assert.Error(t, err)
}

func Test_BadDelete(t *testing.T) {
	t.Skip()
	o := &topoapi.Object{ID: "nothing", Type: topoapi.Object_ENTITY, Obj: &topoapi.Object_Entity{Entity: &topoapi.Entity{}}}
	err := store.Delete(context.TODO(), o)
	assert.Error(t, err)
}

func Test_BadGet(t *testing.T) {
	t.Skip()
	o, err := store.Get(context.TODO(), topoapi.ID("nothing"))
	assert.Error(t, err)
	assert.Nil(t, o)
}
