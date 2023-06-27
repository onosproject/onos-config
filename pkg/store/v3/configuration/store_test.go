// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"context"
	"github.com/atomix/go-sdk/pkg/test"
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v3"
	"github.com/stretchr/testify/assert"
)

func TestConfigurationStore(t *testing.T) {
	cluster := test.NewClient()
	defer cluster.Close()

	store1, err := NewAtomixStore(cluster)
	assert.NoError(t, err)

	store2, err := NewAtomixStore(cluster)
	assert.NoError(t, err)

	target1 := configapi.Target{
		ID:      "target-1",
		Type:    "foo",
		Version: "1",
	}
	target2 := configapi.Target{
		ID:      "target-2",
		Type:    "bar",
		Version: "1",
	}

	ch := make(chan configapi.ConfigurationEvent)
	err = store2.Watch(context.Background(), ch)
	assert.NoError(t, err)

	target1ConfigValues := make(map[string]configapi.PathValue)
	target1ConfigValues["/foo"] = configapi.PathValue{
		Path: "/foo",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world!"),
			Type:  configapi.ValueType_STRING,
		},
	}

	target1Config := &configapi.Configuration{
		ID: configapi.ConfigurationID{
			Target: target1,
		},
	}
	target1Config.Committed.Values = target1ConfigValues

	target2ConfigValues := make(map[string]configapi.PathValue)
	target2ConfigValues["/foo"] = configapi.PathValue{
		Path: "bar",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world again!"),
			Type:  configapi.ValueType_STRING,
		},
	}
	target2Config := &configapi.Configuration{
		ID: configapi.ConfigurationID{
			Target: target2,
		},
	}
	target2Config.Committed.Values = target2ConfigValues

	err = store1.Create(context.TODO(), target1Config)
	assert.NoError(t, err)
	assert.Equal(t, target1, target1Config.ID.Target)
	assert.NotEqual(t, configapi.Revision(0), target1Config.Revision)

	err = store2.Create(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.Equal(t, target2, target2Config.ID.Target)
	assert.NotEqual(t, configapi.Revision(0), target2Config.Revision)

	// Get the configuration
	target1Config, err = store2.Get(context.TODO(), configapi.ConfigurationID{Target: target1})
	assert.NoError(t, err)
	assert.NotNil(t, target1Config)
	assert.Equal(t, target1, target1Config.ID.Target)
	assert.NotEqual(t, configapi.Revision(0), target1Config.Revision)

	// Verify events were received for the configurations
	configurationEvent := nextEvent(t, ch)
	assert.NotNil(t, configurationEvent)
	configurationEvent = nextEvent(t, ch)
	assert.NotNil(t, configurationEvent)

	// Watch events for a specific configuration
	configurationCh := make(chan configapi.ConfigurationEvent)
	err = store1.Watch(context.TODO(), configurationCh, WithConfigurationID(target2Config.ID))
	assert.NoError(t, err)

	// Update one of the configurations
	revision := target2Config.Revision
	err = store1.Update(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, target2Config.Revision)

	event := <-configurationCh
	assert.Equal(t, target2Config.ID, event.Configuration.ID)

	// Lists configurations
	configurationList, err := store1.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(configurationList))

	// Read and then update the configuration
	target2Config, err = store2.Get(context.TODO(), configapi.ConfigurationID{Target: target2})
	assert.NoError(t, err)
	assert.NotNil(t, target2Config)
	target2Config.Status.State = configapi.ConfigurationStatus_SYNCHRONIZED
	revision = target2Config.Revision
	err = store1.Update(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, target2Config.Revision)

	event = <-configurationCh
	assert.Equal(t, target2Config.ID, event.Configuration.ID)

	// Verify that concurrent updates fail
	target1Config11, err := store1.Get(context.TODO(), configapi.ConfigurationID{Target: target1})
	assert.NoError(t, err)
	target1Config12, err := store2.Get(context.TODO(), configapi.ConfigurationID{Target: target1})
	assert.NoError(t, err)

	target1Config11.Status.State = configapi.ConfigurationStatus_SYNCHRONIZED
	err = store1.Update(context.TODO(), target1Config11)
	assert.NoError(t, err)

	target1Config12.Status.State = configapi.ConfigurationStatus_SYNCHRONIZING
	err = store2.Update(context.TODO(), target1Config12)
	assert.Error(t, err)

	// Verify events were received again
	configurationEvent = nextEvent(t, ch)
	assert.NotNil(t, configurationEvent)
	configurationEvent = nextEvent(t, ch)
	assert.NotNil(t, configurationEvent)
	configurationEvent = nextEvent(t, ch)
	assert.NotNil(t, configurationEvent)

	// Checks list of configuration after deleting a configuration
	configurationList, err = store2.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(configurationList))

	err = store1.Close(context.TODO())
	assert.NoError(t, err)

	err = store2.Close(context.TODO())
	assert.NoError(t, err)

}

func nextEvent(t *testing.T, ch chan configapi.ConfigurationEvent) *configapi.Configuration {
	select {
	case c := <-ch:
		return &c.Configuration
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
