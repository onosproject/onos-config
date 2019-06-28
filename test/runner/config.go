package runner

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

var (
	_, path, _, _     = runtime.Caller(0)
	certsPath         = filepath.Join(filepath.Dir(filepath.Dir(path)), "certs")
	deviceConfigsPath = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "device")
	storeConfigsPath  = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "store")
)

// ClusterConfig provides the configuration for the Kubernetes test cluster
type ClusterConfig struct {
	Preset        string `yaml:"preset" mapstructure:"preset"`
	Nodes         int    `yaml:"nodes" mapstructure:"nodes"`
	Partitions    int    `yaml:"partitions" mapstructure:"partitions"`
	PartitionSize int    `yaml:"partitionSize" mapstructure:"partitionSize"`
}

// load loads the preset configuration for the cluster
func (c *ClusterConfig) load() (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(storeConfigsPath, c.Preset+".json"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	jsonBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var jsonObj map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonObj)
	if err != nil {
		return nil, err
	}
	return jsonObj, nil
}

// SimulatorConfig provides the configuration for a device simulator
type SimulatorConfig struct {
	Config string `yaml:"config" mapstructure:"config"`
}

// load loads the simulator configuration
func (c *SimulatorConfig) load() (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(deviceConfigsPath, c.Config+".json"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	jsonBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var jsonObj map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonObj)
	return jsonObj, err
}
