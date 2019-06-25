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

package runner

import (
	"fmt"
	atomixk8s "github.com/atomix/atomix-k8s-controller/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	log "k8s.io/klog"
	"time"
)

// GetClusterController returns a Kubernetes integration test controller for the given test ID
func GetClusterController(clusterId string, config *ClusterConfig) (*ClusterController, error) {
	setClusterConfigDefaults(config)

	kubeclient, err := newKubeClient()
	if err != nil {
		return nil, err
	}

	atomixclient, err := newAtomixKubeClient()
	if err != nil {
		return nil, err
	}

	extensionsclient, err := newExtensionsKubeClient()
	if err != nil {
		return nil, err
	}

	return &ClusterController{
		ClusterId:        clusterId,
		kubeclient:       kubeclient,
		atomixclient:     atomixclient,
		extensionsclient: extensionsclient,
		config:           config,
	}, nil
}

// Kubernetes cluster controller
type ClusterController struct {
	ClusterId        string
	kubeclient       *kubernetes.Clientset
	atomixclient     *atomixk8s.Clientset
	extensionsclient *apiextension.Clientset
	config           *ClusterConfig
}

// getClusterName returns the qualified cluster name
func (c *ClusterController) getClusterName() string {
	return fmt.Sprintf("onos-cluster-%s", c.ClusterId)
}

// SetupCluster sets up a test cluster with the given configuration
func (c *ClusterController) SetupCluster() error {
	log.Infof("Setting up test cluster %s", c.ClusterId)
	if err := c.setupNamespace(); err != nil {
		return err
	}
	if err := c.setupAtomixController(); err != nil {
		return err
	}
	if err := c.setupPartitions(); err != nil {
		return err
	}
	if err := c.setupOnosConfig(); err != nil {
		return err
	}
	return nil
}

// SetupSimulator sets up a device simulator with the given configuration
func (c *ClusterController) SetupSimulator(name string, config *SimulatorConfig) error {
	log.Infof("Setting up simulator %s/%s", name, c.getClusterName())
	if err := c.setupSimulator(name, config); err != nil {
		return err
	}

	log.Infof("Waiting for simulator %s/%s to become ready", name, c.getClusterName())
	if err := c.awaitSimulatorReady(name); err != nil {
		return err
	}
	return c.redeployOnosConfig()
}

// RunTests runs the given tests on Kubernetes
func (c *ClusterController) RunTests(testId string, tests []string, timeout time.Duration) (string, int, error) {
	// Default the test timeout to 10 minutes
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	// Start the test job
	pod, err := c.startTests(testId, tests, timeout)
	if err != nil {
		return "", 0, err
	}

	// Stream the logs to stdout
	if err = c.streamLogs(pod); err != nil {
		return "", 0, err
	}

	// Get the exit message and code
	return c.getStatus(pod)
}

// TeardownSimulator tears down a device simulator with the given name
func (c *ClusterController) TeardownSimulator(name string) error {
	log.Infof("Tearing down simulator %s/%s", name, c.getClusterName())
	if err := c.teardownSimulator(name); err != nil {
		return err
	}
	return c.redeployOnosConfig()
}

// TeardownCluster tears down the test cluster
func (c *ClusterController) TeardownCluster() error {
	log.Infof("Tearing down test namespace %s", c.getClusterName())
	if err := c.deleteNamespace(); err != nil {
		return err
	}
	if err := c.deleteClusterRoleBinding(); err != nil {
		return err
	}
	return nil
}

// setupNamespace creates a uniquely named namespace with which to run tests
func (c *ClusterController) setupNamespace() error {
	log.Infof("Setting up test namespace %s", c.getClusterName())
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.getClusterName(),
		},
	}
	_, err := c.kubeclient.CoreV1().Namespaces().Create(namespace)
	return err
}

// deleteClusterRoleBinding deletes the ClusterRoleBinding used by the test
func (c *ClusterController) deleteClusterRoleBinding() error {
	return c.kubeclient.RbacV1().ClusterRoleBindings().Delete("atomix-controller", &metav1.DeleteOptions{})
}

// deleteNamespace deletes the Namespace used by the test and all resources within it
func (c *ClusterController) deleteNamespace() error {
	return c.kubeclient.CoreV1().Namespaces().Delete(c.getClusterName(), &metav1.DeleteOptions{})
}
