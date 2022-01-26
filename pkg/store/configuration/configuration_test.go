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

package configuration

import (
	"context"
	"testing"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/stretchr/testify/assert"
)

func TestConfigurationStore(t *testing.T) {
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	client1, err := test.NewClient("node-1")
	assert.NoError(t, err)

	client2, err := test.NewClient("node-2")
	assert.NoError(t, err)

	store1, err := NewAtomixStore(client1)
	assert.NoError(t, err)

	store2, err := NewAtomixStore(client2)
	assert.NoError(t, err)

	target1 := configapi.TargetID("target-1")
	target2 := configapi.TargetID("target-2")

	ch := make(chan configapi.ConfigurationEvent)
	err = store2.Watch(context.Background(), ch)
	assert.NoError(t, err)

	target1ConfigValues := make(map[string]*configapi.PathValue)
	target1ConfigValues["/foo"] = &configapi.PathValue{
		Path: "/foo",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world!"),
			Type:  configapi.ValueType_STRING,
		},
	}

	target1Config := &configapi.Configuration{
		ID:            configapi.ConfigurationID(target1),
		TargetID:      target1,
		TargetVersion: "1.0.0",
		Values:        target1ConfigValues,
	}

	target2ConfigValues := make(map[string]*configapi.PathValue)
	target2ConfigValues["/foo"] = &configapi.PathValue{
		Path: "bar",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world again!"),
			Type:  configapi.ValueType_STRING,
		},
	}
	target2Config := &configapi.Configuration{
		ID:            configapi.ConfigurationID(target2),
		TargetID:      target2,
		TargetVersion: "1.0.0",
		Values:        target2ConfigValues,
	}

	err = store1.Create(context.TODO(), target1Config)
	assert.NoError(t, err)
	assert.Equal(t, configapi.ConfigurationID(target1), target1Config.ID)
	assert.NotEqual(t, configapi.Revision(0), target1Config.Revision)

	err = store2.Create(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.Equal(t, configapi.ConfigurationID(target2), target2Config.ID)
	assert.NotEqual(t, configapi.Revision(0), target2Config.Revision)

	// Get the configuration
	target1Config, err = store2.Get(context.TODO(), configapi.ConfigurationID(target1))
	assert.NoError(t, err)
	assert.NotNil(t, target1Config)
	assert.Equal(t, configapi.ConfigurationID(target1), target1Config.ID)
	assert.NotEqual(t, configapi.Revision(0), target1Config.Revision)

	// Verify events were received for the configurations
	configurationEvent := nextEvent(t, ch)
	assert.Equal(t, configapi.ConfigurationID(target1), configurationEvent.ID)
	configurationEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.ConfigurationID(target2), configurationEvent.ID)

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
	assert.Equal(t, target2Config.Revision, event.Configuration.Revision)

	// Lists configurations
	configurationList, err := store1.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(configurationList))

	// Read and then update the configuration
	target2Config, err = store2.Get(context.TODO(), configapi.ConfigurationID(target2))
	assert.NoError(t, err)
	assert.NotNil(t, target2Config)
	target2Config.Status.State = configapi.ConfigurationState_CONFIGURATION_COMPLETE
	revision = target2Config.Revision
	err = store1.Update(context.TODO(), target2Config)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, target2Config.Revision)

	event = <-configurationCh
	assert.Equal(t, target2Config.ID, event.Configuration.ID)
	assert.Equal(t, target2Config.Revision, event.Configuration.Revision)

	// Verify that concurrent updates fail
	target1Config11, err := store1.Get(context.TODO(), configapi.ConfigurationID(target1))
	assert.NoError(t, err)
	target1Config12, err := store2.Get(context.TODO(), configapi.ConfigurationID(target1))
	assert.NoError(t, err)

	target1Config11.Status.State = configapi.ConfigurationState_CONFIGURATION_COMPLETE
	err = store1.Update(context.TODO(), target1Config11)
	assert.NoError(t, err)

	target1Config12.Status.State = configapi.ConfigurationState_CONFIGURATION_PENDING
	err = store2.Update(context.TODO(), target1Config12)
	assert.Error(t, err)

	// Verify events were received again
	configurationEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.ConfigurationID(target2), configurationEvent.ID)
	configurationEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.ConfigurationID(target2), configurationEvent.ID)
	configurationEvent = nextEvent(t, ch)
	assert.Equal(t, configapi.ConfigurationID(target1), configurationEvent.ID)

	// Delete a configuration
	err = store1.Delete(context.TODO(), target2Config)
	assert.NoError(t, err)
	configuration, err := store2.Get(context.TODO(), configapi.ConfigurationID(target2))
	assert.NoError(t, err)
	assert.NotNil(t, configuration.Deleted)
	err = store1.Delete(context.TODO(), target2Config)
	assert.NoError(t, err)
	configuration, err = store2.Get(context.TODO(), configapi.ConfigurationID(target2))
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
	assert.Nil(t, configuration)
	event = <-configurationCh
	assert.Equal(t, target2Config.ID, event.Configuration.ID)
	assert.Equal(t, configapi.ConfigurationEventType_CONFIGURATION_UPDATED, event.Type)
	event = <-configurationCh
	assert.Equal(t, target2Config.ID, event.Configuration.ID)
	assert.Equal(t, configapi.ConfigurationEventType_CONFIGURATION_DELETED, event.Type)

	// Checks list of configuration after deleting a configuration
	configurationList, err = store2.List(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configurationList))

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
