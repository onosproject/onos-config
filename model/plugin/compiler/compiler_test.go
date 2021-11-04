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

package plugincompiler

import (
	"context"
	"github.com/onosproject/onos-config/model"
	plugincache "github.com/onosproject/onos-config/model/plugin/cache"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	_, b, _, _ = runtime.Caller(0)
	moduleRoot = filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(b))))
)

func TestCompiler(t *testing.T) {
	cache, err := plugincache.NewPluginCache(plugincache.CacheConfig{
		Path: filepath.Join(moduleRoot, "test", "model", "cache"),
	})
	assert.NoError(t, err)

	config := CompilerConfig{
		TemplatePath: filepath.Join(moduleRoot, "model", "plugin", "compiler", "templates"),
		BuildPath:    moduleRoot,
	}

	bytes, err := ioutil.ReadFile(filepath.Join(moduleRoot, "test", "model", "test@2020-11-18.yang"))
	assert.NoError(t, err)

	modelInfo := configmodel.ModelInfo{
		Name:         "test",
		Version:      "1.0.0",
		GetStateMode: configmodel.GetStateExplicitRoPaths,
		Modules: []configmodel.ModuleInfo{
			{
				Name:         "test",
				Organization: "ONF",
				Revision:     "2020-11-18",
				File:         "test.yang",
			},
		},
		Files: []configmodel.FileInfo{
			{
				Path: "test@2020-11-18.yang",
				Data: bytes,
			},
		},
		Plugin: configmodel.PluginInfo{
			Name:    "test",
			Version: "1.0.0",
		},
	}

	entry := cache.Entry("test", "1.0.0")
	err = entry.Lock(context.TODO())
	assert.NoError(t, err)

	compiler := NewPluginCompiler(config)
	err = compiler.CompilePlugin(modelInfo, entry.Path)
	assert.NoError(t, err)

	plugin, err := entry.Load()
	assert.NoError(t, err)
	assert.NotNil(t, plugin)

	err = entry.Unlock(context.TODO())
	assert.NoError(t, err)
}
