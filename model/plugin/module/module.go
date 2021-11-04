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

package module

import (
	"crypto/md5"
	"github.com/rogpeppe/go-internal/modfile"
	"io/ioutil"
	"path/filepath"
)

const defaultPath = "/etc/onos/mod"

type Config struct {
	Path string `json:"path"`
}

func NewModule(config Config) *Module {
	if config.Path == "" {
		config.Path = defaultPath
	}
	return &Module{
		Config: config,
	}
}

type Hash []byte

type Module struct {
	Config
}

func (m *Module) Init() error {
	modHashPath := filepath.Join(m.Path, "mod.hash")
	_, err := ioutil.ReadFile(modHashPath)
	if err == nil {
		return nil
	}

	modPath := filepath.Join(m.Path, "go.mod")
	modBytes, err := ioutil.ReadFile(modPath)
	if err != nil {
		return err
	}

	mod, err := modfile.Parse(modPath, modBytes, nil)
	if err != nil {
		return err
	}

	modBytes, err = mod.Format()
	if err != nil {
		return err
	}
	modHashBytes := md5.Sum(modBytes)
	modHash := modHashBytes[:]

	err = ioutil.WriteFile(modHashPath, modHash, 0666)
	if err != nil {
		return err
	}
	return nil
}

func (m *Module) GetHash() (Hash, error) {
	modHashPath := filepath.Join(m.Path, "mod.hash")
	modHash, err := ioutil.ReadFile(modHashPath)
	if err != nil {
		return nil, err
	}
	return modHash, nil
}
