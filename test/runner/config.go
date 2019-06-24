package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gofrs/flock"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

const (
	defaultClusterKey = "default"
	clustersKey       = "clusters"
)

var (
	_, path, _, _     = runtime.Caller(0)
	certsPath         = filepath.Join(filepath.Dir(filepath.Dir(path)), "certs")
	deviceConfigsPath = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "device")
	storeConfigsPath  = filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(path)), "configs"), "store")
)

var (
	configLock *flock.Flock
)

func init() {
	cobra.OnInitialize(initConfig)
}

// getSimulatorPreset gets a device configuration by name
func getSimulatorPreset(name string) (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(deviceConfigsPath, name+".json"))
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

// getSimulatorPresets returns a list of store configurations from the configs/store directory
func getSimulatorPresets() []string {
	configs := []string{}
	filepath.Walk(deviceConfigsPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		if ext == ".json" {
			configs = append(configs, info.Name()[:len(info.Name())-len(ext)])
		}
		return nil
	})
	return configs
}

// getStorePreset returns a named store configuration from the given stores configuration
func getStorePreset(configName string, storeType string) (map[string]interface{}, error) {
	file, err := os.Open(filepath.Join(storeConfigsPath, configName+".json"))
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

	storeObj, ok := jsonObj[storeType]
	if !ok {
		return nil, errors.New("malformed store configuration: " + storeType + " key not found")
	}

	return storeObj.(map[string]interface{}), nil
}

// getChangeStorePreset returns the change store from the given stores configuration
func getChangeStorePreset(name string) (map[string]interface{}, error) {
	return getStorePreset(name, "changeStore")
}

// getConfigStorePreset returns the config store from the given stores configuration
func getConfigStorePreset(name string) (map[string]interface{}, error) {
	return getStorePreset(name, "configStore")
}

// getNetworkStorePreset returns the network store from the given stores configuration
func getNetworkStorePreset(name string) (map[string]interface{}, error) {
	return getStorePreset(name, "networkStore")
}

// getDeviceStorePreset returns the device store from the given stores configuration
func getDeviceStorePreset(name string) (map[string]interface{}, error) {
	return getStorePreset(name, "deviceStore")
}

// getStorePresets returns a list of store configurations from the configs/store directory
func getStorePresets() []string {
	configs := []string{}
	filepath.Walk(storeConfigsPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		if ext == ".json" {
			configs = append(configs, info.Name()[:len(info.Name())-len(ext)])
		}
		return nil
	})
	return configs
}

func addCluster(name string, cluster *ClusterConfig) error {
	lock, err := getConfigLock()
	if err != nil {
		return err
	}
	if err = lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()

	config, err := LoadConfig()
	if err != nil {
		return err
	}

	config.DefaultCluster = name
	config.Clusters[name] = cluster
	return config.Write()
}

func removeCluster(name string) error {
	lock, err := getConfigLock()
	if err != nil {
		return err
	}
	if err = lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()

	config, err := LoadConfig()
	if err != nil {
		return err
	}

	delete(config.Clusters, name)
	if config.DefaultCluster == name {
		config.DefaultCluster = ""
	}
	return config.Write()
}

func addSimulator(cluster string, name string, simulator *SimulatorConfig) error {
	lock, err := getConfigLock()
	if err != nil {
		return err
	}
	if err = lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()

	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Fail if the cluster does not exist
	clusterConfig, ok := config.Clusters[cluster]
	if !ok {
		return errors.New("unknown cluster " + cluster)
	}

	// Fail if the simulator already exists
	_, ok = clusterConfig.Simulators[name]
	if ok {
		return errors.New("simulator " + name + " already exists")
	}

	clusterConfig.Simulators[name] = simulator
	return config.Write()
}

func removeSimulator(cluster string, name string) error {
	lock, err := getConfigLock()
	if err != nil {
		return err
	}
	if err = lock.Lock(); err != nil {
		return err
	}
	defer lock.Unlock()

	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Fail if the cluster does not exist
	clusterConfig, ok := config.Clusters[cluster]
	if !ok {
		return errors.New("unknown cluster " + cluster)
	}

	// Fail if the simulator does not exist
	_, ok = clusterConfig.Simulators[name]
	if !ok {
		return errors.New("unknown simulator " + name)
	}

	delete(clusterConfig.Simulators, name)
	return config.Write()
}

// getConfigLock returns a configuration file lock
func getConfigLock() (*flock.Flock, error) {
	_, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	configFile := viper.ConfigFileUsed()
	lockFile := configFile[0:len(configFile)-len(filepath.Ext(configFile))] + ".lock"
	return flock.New(lockFile), nil
}

// LoadConfig loads the onit configuration from a configuration file
func LoadConfig() (*OnitConfig, error) {
	config := &OnitConfig{Clusters: make(map[string]*ClusterConfig)}
	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}
	return config, nil
}

// OnitConfig provides the configuration for onit
type OnitConfig struct {
	DefaultCluster string                    `yaml:"default" mapstructure:"default"`
	Clusters       map[string]*ClusterConfig `yaml:"clusters" mapstructure:"clusters"`
}

// getDefaultCluster returns the default cluster ID
func (c *OnitConfig) getDefaultCluster() (string, error) {
	cluster := c.DefaultCluster
	if cluster == "" {
		return "", errors.New("no default cluster set")
	}
	return cluster, nil
}

// getDefaultClusterConfig returns the default cluster configuration
func (c *OnitConfig) getDefaultClusterConfig() (*ClusterConfig, error) {
	cluster, err := c.getDefaultCluster()
	if err != nil {
		return nil, err
	}

	config, ok := c.Clusters[cluster]
	if !ok {
		return nil, errors.New("unknown default cluster " + cluster)
	}
	return config, nil
}

// getClusterConfig returns the configuration for the given cluster
func (c *OnitConfig) getClusterConfig(clusterId string) (*ClusterConfig, error) {
	config, ok := c.Clusters[clusterId]
	if !ok {
		return nil, errors.New("unknown default cluster " + clusterId)
	}
	return config, nil
}

// Lock acquires a lock on the configuration
func (c *OnitConfig) Lock() error {
	// Get the configuration file used by viper and create a lock file from it
	configFile := viper.ConfigFileUsed()
	lockFile := configFile[0:len(configFile)-len(filepath.Ext(configFile))] + ".lock"
	configLock := flock.New(lockFile)

	// Acquire the lock on the configuration file
	if err := configLock.Lock(); err != nil {
		return err
	}

	// Once the lock has been acquired, load the configuration from the file to ensure it's up-to-date
	return viper.Unmarshal(c)
}

// Unlock releases a lock on the configuration
func (c *OnitConfig) Unlock() error {
	lock := configLock
	if lock == nil {
		return errors.New("no configuration lock acquired")
	}
	return lock.Unlock()
}

// Write writes the configuration to a configuration file
func (c *OnitConfig) Write() error {
	encoded, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	if err = viper.ReadConfig(bytes.NewReader(encoded)); err != nil {
		return err
	}
	return viper.WriteConfig()
}

// ClusterConfig provides the configuration for the Kubernetes test cluster
type ClusterConfig struct {
	ChangeStore   map[string]interface{}      `yaml:"changeStore" mapstructure:"changeStore"`
	ConfigStore   map[string]interface{}      `yaml:"configStore" mapstructure:"configStore"`
	DeviceStore   map[string]interface{}      `yaml:"deviceStore" mapstructure:"deviceStore"`
	NetworkStore  map[string]interface{}      `yaml:"networkStore" mapstructure:"networkStore"`
	Simulators    map[string]*SimulatorConfig `yaml:"simulators" mapstructure:"simulators"`
	Nodes         int                         `yaml:"nodes" mapstructure:"nodes"`
	Partitions    int                         `yaml:"partitions" mapstructure:"partitions"`
	PartitionSize int                         `yaml:"partitionSize" mapstructure:"partitionSize"`
}

// getSimulator returns a simulator configuration
func (c *ClusterConfig) getSimulator(name string) (*SimulatorConfig, error) {
	simulator, ok := c.Simulators[name]
	if !ok {
		return nil, errors.New("unknown simulator " + name)
	}
	return simulator, nil
}

func setClusterConfigDefaults(config *ClusterConfig) {
	if config.ChangeStore == nil {
		config.ChangeStore = map[string]interface{}{
			"Version":   "1.0.0",
			"Storetype": "change",
			"Store":     make(map[string]interface{}),
		}
	}
	if config.DeviceStore == nil {
		config.DeviceStore = map[string]interface{}{
			"Version":   "1.0.0",
			"Storetype": "device",
			"Store":     make(map[string]interface{}),
		}
	}
	if config.ConfigStore == nil {
		config.ConfigStore = map[string]interface{}{
			"Version":   "1.0.0",
			"Storetype": "config",
			"Store":     make(map[string]interface{}),
		}
	}
	if config.NetworkStore == nil {
		config.NetworkStore = map[string]interface{}{
			"Version":   "1.0.0",
			"Storetype": "network",
			"Store":     make([]interface{}, 0),
		}
	}
	if config.Simulators == nil {
		config.Simulators = make(map[string]*SimulatorConfig)
	}
	if config.Nodes == 0 {
		config.Nodes = 1
	}
	if config.Partitions == 0 {
		config.Partitions = 1
	}
	if config.PartitionSize == 0 {
		config.PartitionSize = 1
	}
}

// SimulatorConfig provides the configuration for a device simulator
type SimulatorConfig struct {
	Config map[string]interface{} `yaml:"config" mapstructure:"config"`
}

func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		exitError(err)
	}

	viper.SetConfigName("onit")
	viper.AddConfigPath(home + "/.onos")
	viper.AddConfigPath("/etc/onos")
	viper.AddConfigPath(".")

	// If the configuration file is not found, initialize a configuration in the home dir.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(*viper.ConfigFileNotFoundError); !ok {
			home, err := homedir.Dir()
			if err != nil {
				exitError(err)
			}

			err = os.MkdirAll(home+"/.onos", 0777)
			if err != nil {
				exitError(err)
			}

			f, err := os.Create(home + "/.onos/onit.yaml")
			if err != nil {
				exitError(err)
			} else {
				f.Close()
			}

			err = viper.WriteConfig()
			if err != nil {
				exitError(err)
			}
		} else {
			exitError(err)
		}
	}
}
