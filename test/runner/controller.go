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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/atomix/atomix-k8s-controller/pkg/apis/k8s/v1alpha1"
	atomixk8s "github.com/atomix/atomix-k8s-controller/pkg/client/clientset/versioned"
	raft "github.com/atomix/atomix-k8s-controller/proto/atomix/protocols/raft"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/onosproject/onos-config/test/env"
	"io"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	log "k8s.io/klog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TestClusterConfig provides the configuration for the Kubernetes test cluster
type TestClusterConfig struct {
	ChangeStore   map[string]interface{}
	ConfigStore   map[string]interface{}
	DeviceStore   map[string]interface{}
	NetworkStore  map[string]interface{}
	Simulators    map[string]interface{}
	Nodes         int
	Partitions    int
	PartitionSize int
}

// TestSimulatorConfig provides the configuration for a device simulator
type TestSimulatorConfig struct {
	Config map[string]interface{}
}

// NewTestController creates a new Kubernetes integration test controller
func NewTestController(config *TestClusterConfig) (*TestController, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	return GetTestController(id.String(), config)
}

// GetTestController returns a Kubernetes integration test controller for the given test ID
func GetTestController(testId string, config *TestClusterConfig) (*TestController, error) {
	testName := getTestName(testId)
	setTestClusterConfigDefaults(config)

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

	return &TestController{
		TestId:           testId,
		TestName:         testName,
		kubeclient:       kubeclient,
		atomixclient:     atomixclient,
		extensionsclient: extensionsclient,
		config:           config,
	}, nil
}

func setTestClusterConfigDefaults(config *TestClusterConfig) {
	if config.ChangeStore == nil {
		config.ChangeStore = make(map[string]interface{})
	}
	if config.DeviceStore == nil {
		config.DeviceStore = make(map[string]interface{})
	}
	if config.ConfigStore == nil {
		config.ConfigStore = make(map[string]interface{})
	}
	if config.NetworkStore == nil {
		config.NetworkStore = make(map[string]interface{})
	}
	if config.Simulators == nil {
		config.Simulators = make(map[string]interface{})
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

// Kubernetes test controller
type TestController struct {
	TestId           string
	TestName         string
	kubeclient       *kubernetes.Clientset
	atomixclient     *atomixk8s.Clientset
	extensionsclient *apiextension.Clientset
	config           *TestClusterConfig
}

// SetupCluster sets up a test cluster with the given configuration
func (c *TestController) SetupCluster() error {
	log.Infof("Setting up test cluster %s", c.TestId)
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
func (c *TestController) SetupSimulator(name string, config *TestSimulatorConfig) error {
	log.Infof("Setting up simulator %s/%s", name, c.TestName)
	if err := c.setupSimulator(name, config); err != nil {
		return err
	}

	log.Infof("Waiting for simulator %s/%s to become ready", name, c.TestName)
	if err := c.awaitSimulatorReady(name); err != nil {
		return err
	}
	return c.redeployOnosConfig()
}

// RunTests runs the given tests on Kubernetes
func (c *TestController) RunTests(tests []string, timeout time.Duration) (string, int, error) {
	// Default the test timeout to 10 minutes
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	// Start the test job
	pod, err := c.start(tests, timeout)
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
func (c *TestController) TeardownSimulator(name string) error {
	log.Infof("Tearing down simulator %s/%s", name, c.TestName)
	if err := c.teardownSimulator(name); err != nil {
		return err
	}
	return c.redeployOnosConfig()
}

// TeardownCluster tears down the test cluster
func (c *TestController) TeardownCluster() error {
	log.Infof("Tearing down test namespace %s", c.TestName)
	if err := c.deleteNamespace(); err != nil {
		return err
	}
	if err := c.deleteClusterRoleBinding(); err != nil {
		return err
	}
	return nil
}

// setupNamespace creates a uniquely named namespace with which to run tests
func (c *TestController) setupNamespace() error {
	log.Infof("Setting up test namespace %s", c.TestName)
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.TestName,
		},
	}
	_, err := c.kubeclient.CoreV1().Namespaces().Create(namespace)
	return err
}

// setupAtomixController sets up the Atomix controller and associated resources
func (c *TestController) setupAtomixController() error {
	log.Infof("Setting up Atomix controller atomix-controller/%s", c.TestName)
	if err := c.createAtomixPartitionSetResource(); err != nil {
		return err
	}
	if err := c.createAtomixPartitionResource(); err != nil {
		return err
	}
	if err := c.createAtomixClusterRole(); err != nil {
		return err
	}
	if err := c.createAtomixClusterRoleBinding(); err != nil {
		return err
	}
	if err := c.createAtomixServiceAccount(); err != nil {
		return err
	}
	if err := c.createAtomixDeployment(); err != nil {
		return err
	}
	if err := c.createAtomixService(); err != nil {
		return err
	}

	log.Infof("Waiting for Atomix controller atomix-controller/%s to become ready", c.TestName)
	if err := c.awaitAtomixControllerReady(); err != nil {
		return err
	}
	return nil
}

// createAtomixPartitionSetResource creates the PartitionSet custom resource definition in the k8s cluster
func (c *TestController) createAtomixPartitionSetResource() error {
	crd := &apiextensionv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "partitionsets.k8s.atomix.io",
		},
		Spec: apiextensionv1beta1.CustomResourceDefinitionSpec{
			Group: "k8s.atomix.io",
			Names: apiextensionv1beta1.CustomResourceDefinitionNames{
				Kind:     "PartitionSet",
				ListKind: "PartitionSetList",
				Plural:   "partitionsets",
				Singular: "partitionset",
			},
			Scope:   apiextensionv1beta1.NamespaceScoped,
			Version: "v1alpha1",
			Subresources: &apiextensionv1beta1.CustomResourceSubresources{
				Status: &apiextensionv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}

	_, err := c.extensionsclient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createAtomixPartitionResource creates the Partition custom resource definition in the k8s cluster
func (c *TestController) createAtomixPartitionResource() error {
	crd := &apiextensionv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "partitions.k8s.atomix.io",
		},
		Spec: apiextensionv1beta1.CustomResourceDefinitionSpec{
			Group: "k8s.atomix.io",
			Names: apiextensionv1beta1.CustomResourceDefinitionNames{
				Kind:     "Partition",
				ListKind: "PartitionList",
				Plural:   "partitions",
				Singular: "partition",
			},
			Scope:   apiextensionv1beta1.NamespaceScoped,
			Version: "v1alpha1",
			Subresources: &apiextensionv1beta1.CustomResourceSubresources{
				Status: &apiextensionv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}

	_, err := c.extensionsclient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createAtomixClusterRole creates the ClusterRole required by the Atomix controller if not yet created
func (c *TestController) createAtomixClusterRole() error {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods",
					"services",
					"endpoints",
					"persistentvolumeclaims",
					"events",
					"configmaps",
					"secrets",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"namespaces",
				},
				Verbs: []string{
					"get",
				},
			},
			{
				APIGroups: []string{
					"apps",
				},
				Resources: []string{
					"deployments",
					"daemonsets",
					"replicasets",
					"statefulsets",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"poddisruptionbudgets",
				},
				Verbs: []string{
					"*",
				},
			},
			{
				APIGroups: []string{
					"k8s.atomix.io",
				},
				Resources: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			},
		},
	}
	_, err := c.kubeclient.RbacV1().ClusterRoles().Create(role)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// createAtomixClusterRoleBinding creates the ClusterRoleBinding required by the Atomix controller for the test namespace
func (c *TestController) createAtomixClusterRoleBinding() error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "atomix-controller",
				Namespace: c.TestName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "atomix-controller",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	_, err := c.kubeclient.RbacV1().ClusterRoleBindings().Create(roleBinding)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			c.deleteClusterRoleBinding()
			return c.createAtomixClusterRoleBinding()
		} else {
			return err
		}
	}
	return nil
}

// createAtomixServiceAccount creates a ServiceAccount used by the Atomix controller
func (c *TestController) createAtomixServiceAccount() error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
		},
	}
	_, err := c.kubeclient.CoreV1().ServiceAccounts(c.TestName).Create(serviceAccount)
	return err
}

// createAtomixDeployment creates the Atomix controller Deployment
func (c *TestController) createAtomixDeployment() error {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "atomix-controller",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": "atomix-controller",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "atomix-controller",
					Containers: []corev1.Container{
						{
							Name:            "atomix-controller",
							Image:           "atomix/atomix-k8s-controller:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"atomix-controller"},
							Env: []corev1.EnvVar{
								{
									Name:  "CONTROLLER_NAME",
									Value: "atomix-controller",
								},
								{
									Name: "CONTROLLER_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "control",
									ContainerPort: 5679,
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"stat",
											"/tmp/atomix-controller-ready",
										},
									},
								},
								InitialDelaySeconds: int32(4),
								PeriodSeconds:       int32(10),
								FailureThreshold:    int32(1),
							},
						},
					},
				},
			},
		},
	}
	_, err := c.kubeclient.AppsV1().Deployments(c.TestName).Create(deployment)
	return err
}

// createAtomixService creates a service for the controller
func (c *TestController) createAtomixService() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"name": "atomix-controller",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "control",
					Port: 5679,
				},
			},
		},
	}
	_, err := c.kubeclient.CoreV1().Services(c.TestName).Create(service)
	return err
}

// awaitAtomixControllerReady blocks until the Atomix controller is ready
func (c *TestController) awaitAtomixControllerReady() error {
	for {
		dep, err := c.kubeclient.AppsV1().Deployments(c.TestName).Get("atomix-controller", metav1.GetOptions{})
		if err != nil {
			return err
		} else if dep.Status.ReadyReplicas == 1 {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// setupPartitions creates a Raft partition set
func (c *TestController) setupPartitions() error {
	log.Infof("Setting up partitions raft/%s", c.TestName)
	if err := c.createPartitionSet(); err != nil {
		return err
	}

	log.Infof("Waiting for partitions raft/%s to become ready", c.TestName)
	if err := c.awaitPartitionsReady(); err != nil {
		return err
	}
	return nil
}

// createPartitionSet creates a Raft partition set from the configuration
func (c *TestController) createPartitionSet() error {
	bytes, err := yaml.Marshal(&raft.RaftProtocol{})
	if err != nil {
		return err
	}

	set := &v1alpha1.PartitionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "raft",
			Namespace: c.TestName,
		},
		Spec: v1alpha1.PartitionSetSpec{
			Partitions: c.config.Partitions,
			Template: v1alpha1.PartitionTemplateSpec{
				Spec: v1alpha1.PartitionSpec{
					Size:     int32(c.config.PartitionSize),
					Protocol: "raft",
					Image:    "atomix/atomix-raft-protocol:latest",
					Config:   string(bytes),
				},
			},
		},
	}
	_, err = c.atomixclient.K8sV1alpha1().PartitionSets(c.TestName).Create(set)
	return err
}

// awaitPartitionsReady waits for Raft partitions to complete startup
func (c *TestController) awaitPartitionsReady() error {
	for {
		set, err := c.atomixclient.K8sV1alpha1().PartitionSets(c.TestName).Get("raft", metav1.GetOptions{})
		if err != nil {
			return err
		} else if int(set.Status.ReadyPartitions) == set.Spec.Partitions {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// getDeviceIds returns a slice of configured simulator device IDs
func (c *TestController) getDeviceIds() []string {
	devices := []string{}
	for name, _ := range c.config.DeviceStore {
		devices = append(devices, name)
	}
	for name, _ := range c.config.Simulators {
		devices = append(devices, name)
	}
	return devices
}

// setupSimulator creates a simulator required for the test
func (c *TestController) setupSimulator(name string, config *TestSimulatorConfig) error {
	if err := c.createSimulatorConfigMap(name, config); err != nil {
		return err
	}
	if err := c.createSimulatorPod(name); err != nil {
		return err
	}
	if err := c.createSimulatorService(name); err != nil {
		return err
	}
	return nil
}

// createSimulatorConfigMap creates a simulator configuration
func (c *TestController) createSimulatorConfigMap(name string, config *TestSimulatorConfig) error {
	configJson, err := json.Marshal(config)
	if err != nil {
		return err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.TestName,
		},
		Data: map[string]string{
			"config.json": string(configJson),
		},
	}
	_, err = c.kubeclient.CoreV1().ConfigMaps(c.TestName).Create(cm)
	return err
}

// createSimulatorPod creates a simulator pod
func (c *TestController) createSimulatorPod(name string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.TestName,
			Labels: map[string]string{
				"simulator": name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "device-simulator",
					Image:           "onosproject/device-simulator:latest",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports: []corev1.ContainerPort{
						{
							Name:          "gnmi",
							ContainerPort: 10161,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt(10161),
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt(10161),
							},
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       20,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "config",
							MountPath: "/etc/simulator/configs",
							ReadOnly:  true,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: name,
							},
						},
					},
				},
			},
		},
	}
	_, err := c.kubeclient.CoreV1().Pods(c.TestName).Create(pod)
	return err
}

// createSimulatorService creates a simulator service
func (c *TestController) createSimulatorService(name string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.TestName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"simulator": name,
			},
			Ports: []corev1.ServicePort{
				{
					Name: "gnmi",
					Port: 10161,
				},
			},
		},
	}
	_, err := c.kubeclient.CoreV1().Services(c.TestName).Create(service)
	return err
}

// awaitSimulatorReady waits for the given simulator to complete startup
func (c *TestController) awaitSimulatorReady(name string) error {
	for {
		pod, err := c.kubeclient.CoreV1().Pods(c.TestName).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		} else if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// teardownSimulator tears down a simulator by name
func (c *TestController) teardownSimulator(name string) error {
	if err := c.deleteSimulatorPod(name); err != nil {
		return err
	}
	if err := c.deleteSimulatorService(name); err != nil {
		return err
	}
	if err := c.deleteSimulatorConfigMap(name); err != nil {
		return err
	}
	return nil
}

// deleteSimulatorConfigMap deletes a simulator ConfigMap by name
func (c *TestController) deleteSimulatorConfigMap(name string) error {
	return c.kubeclient.CoreV1().ConfigMaps(c.TestName).Delete(name, &metav1.DeleteOptions{})
}

// deleteSimulatorPod deletes a simulator Pod by name
func (c *TestController) deleteSimulatorPod(name string) error {
	return c.kubeclient.CoreV1().Pods(c.TestName).Delete(name, &metav1.DeleteOptions{})
}

// deleteSimulatorService deletes a simulator Service by name
func (c *TestController) deleteSimulatorService(name string) error {
	return c.kubeclient.CoreV1().Services(c.TestName).Delete(name, &metav1.DeleteOptions{})
}

// setupOnosConfig sets up the onos-config Deployment
func (c *TestController) setupOnosConfig() error {
	log.Infof("Setting up onos-config cluster onos-config/%s", c.TestName)
	if err := c.createOnosConfigSecret(); err != nil {
		return err
	}
	if err := c.createOnosConfigConfigMap(); err != nil {
		return err
	}
	if err := c.createOnosConfigDeployment(); err != nil {
		return err
	}
	if err := c.createOnosConfigService(); err != nil {
		return err
	}

	log.Infof("Waiting for onos-config cluster onos-config/%s to become ready", c.TestName)
	if err := c.awaitOnosConfigDeploymentReady(); err != nil {
		return err
	}
	return nil
}

// createOnosConfigSecret creates a secret for configuring TLS in onos-config and clients
func (c *TestController) createOnosConfigSecret() error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.TestName,
			Namespace: c.TestName,
		},
		StringData: map[string]string{},
	}

	err := filepath.Walk(certsPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}

		secret.StringData[info.Name()] = string(fileBytes)
		return nil
	})
	if err != nil {
		return err
	}

	_, err = c.kubeclient.CoreV1().Secrets(c.TestName).Create(secret)
	return err
}

// createOnosConfigConfigMap creates a ConfigMap for the onos-config Deployment
func (c *TestController) createOnosConfigConfigMap() error {
	// Serialize the change store configuration
	changeStore, err := json.Marshal(c.config.ChangeStore)
	if err != nil {
		return err
	}

	// Serialize the network store configuration
	networkStore, err := json.Marshal(c.config.NetworkStore)
	if err != nil {
		return err
	}

	// If a device store was provided, serialize the device store configuration.
	// Otherwise, create a device store configuration from simulators.
	deviceStoreMap := c.config.DeviceStore
	configStoreMap := c.config.ConfigStore
	for name, _ := range c.config.Simulators {
		deviceMap := make(map[string]interface{})
		deviceMap["ID"] = name
		deviceMap["Addr"] = fmt.Sprintf("%s:10161", name)
		deviceMap["SoftwareVersion"] = "1.0.0"
		deviceMap["Timeout"] = 5
		deviceStoreMap[name] = deviceMap

		configMap := make(map[string]interface{})
		configMap["Name"] = name + "-1.0.0"
		configMap["Device"] = name
		configMap["Version"] = "1.0.0"
		configMap["Type"] = "Devicesim"
		configMap["Created"] = "2019-05-09T16:24:17Z"
		configMap["Updated"] = "2019-05-09T16:24:17Z"
		configMap["Changes"] = []string{}
		configStoreMap[name+"-1.0.0"] = configMap
	}

	// Serialize the device store configuration
	deviceStore, err := json.Marshal(deviceStoreMap)
	if err != nil {
		return err
	}

	// Serialize the config store configuration
	configStore, err := json.Marshal(configStoreMap)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.TestName,
		},
		Data: map[string]string{
			"changeStore.json":  string(changeStore),
			"configStore.json":  string(configStore),
			"deviceStore.json":  string(deviceStore),
			"networkStore.json": string(networkStore),
		},
	}
	_, err = c.kubeclient.CoreV1().ConfigMaps(c.TestName).Create(cm)
	return err
}

// createOnosConfigDeployment creates an onos-config Deployment
func (c *TestController) createOnosConfigDeployment() error {
	nodes := int32(c.config.Nodes)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.TestName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &nodes,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "onos-config",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "onos-config",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "onos-config",
							Image:           "onosproject/onos-config:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "ATOMIX_CONTROLLER",
									Value: fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", c.TestName),
								},
								{
									Name:  "ATOMIX_APP",
									Value: "test",
								},
								{
									Name:  "ATOMIX_NAMESPACE",
									Value: c.TestName,
								},
							},
							Args: []string{
								"-caPath=/etc/onos-config/certs/tls.cacrt",
								"-keyPath=/etc/onos-config/certs/tls.key",
								"-certPath=/etc/onos-config/certs/tls.crt",
								"-configStore=/etc/onos-config/configs/configStore.json",
								"-changeStore=/etc/onos-config/configs/changeStore.json",
								"-deviceStore=/etc/onos-config/configs/deviceStore.json",
								"-networkStore=/etc/onos-config/configs/networkStore.json",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpc",
									ContainerPort: 5150,
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(5150),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt(5150),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/onos-config/configs",
									ReadOnly:  true,
								},
								{
									Name:      "secret",
									MountPath: "/etc/onos-config/certs",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "onos-config",
									},
								},
							},
						},
						{
							Name: "secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: c.TestName,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := c.kubeclient.AppsV1().Deployments(c.TestName).Create(dep)
	return err
}

// createOnosConfigService creates a Service to expose the onos-config Deployment to other pods
func (c *TestController) createOnosConfigService() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.TestName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "onos-config",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "grpc",
					Port: 5150,
				},
			},
		},
	}
	_, err := c.kubeclient.CoreV1().Services(c.TestName).Create(service)
	return err
}

// awaitOnosConfigDeploymentReady waits for the onos-config pods to complete startup
func (c *TestController) awaitOnosConfigDeploymentReady() error {
	for {
		dep, err := c.kubeclient.AppsV1().Deployments(c.TestName).Get("onos-config", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if int(dep.Status.ReadyReplicas) == c.config.Nodes {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// teardownOnosConfig tears down the onos-config deployment
func (c *TestController) redeployOnosConfig() error {
	log.Infof("Redeploying onos-config cluster onos-config/%s", c.TestName)
	if err := c.deleteOnosConfigDeployment(); err != nil {
		return err
	}
	if err := c.deleteOnosConfigConfigMap(); err != nil {
		return err
	}
	if err := c.createOnosConfigConfigMap(); err != nil {
		return err
	}
	if err := c.createOnosConfigDeployment(); err != nil {
		return err
	}
	log.Infof("Waiting for onos-config cluster onos-config/%s to become ready", c.TestName)
	if err := c.awaitOnosConfigDeploymentReady(); err != nil {
		return err
	}
	return nil
}

// deleteOnosConfigConfigMap deletes the onos-config ConfigMap
func (c *TestController) deleteOnosConfigConfigMap() error {
	return c.kubeclient.CoreV1().ConfigMaps(c.TestName).Delete(c.TestName, &metav1.DeleteOptions{})
}

// deleteOnosConfigDeployment deletes the onos-config Deployment
func (c *TestController) deleteOnosConfigDeployment() error {
	return c.kubeclient.AppsV1().Deployments(c.TestName).Delete(c.TestName, &metav1.DeleteOptions{})
}

// start starts running the test job
func (c *TestController) start(args []string, timeout time.Duration) (corev1.Pod, error) {
	if err := c.createTestJob(args, timeout); err != nil {
		return corev1.Pod{}, err
	}
	return c.awaitTestJobRunning()
}

// createTestJob creates the job to run tests
func (c *TestController) createTestJob(args []string, timeout time.Duration) error {
	log.Infof("Starting test job %s", c.TestName)
	one := int32(1)
	timeoutSeconds := int64(timeout / time.Second)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.TestName,
			Namespace: c.TestName,
		},
		Spec: batchv1.JobSpec{
			Parallelism:           &one,
			Completions:           &one,
			BackoffLimit:          &one,
			ActiveDeadlineSeconds: &timeoutSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"test": c.TestName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "test",
							Image:           "onosproject/onos-config-integration-tests:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            args,
							Env: []corev1.EnvVar{
								{
									Name:  env.TestDevicesEnv,
									Value: strings.Join(c.getDeviceIds(), ","),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "secret",
									MountPath: "/etc/onos-config/certs",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: c.TestName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.kubeclient.BatchV1().Jobs(c.TestName).Create(job)
	return err
}

// awaitTestJobRunning blocks until the test job creates a pod in the RUNNING state
func (c *TestController) awaitTestJobRunning() (corev1.Pod, error) {
	log.Infof("Waiting for test job %s to become ready", c.TestName)
	for {
		pods, err := c.kubeclient.CoreV1().Pods(c.TestName).List(metav1.ListOptions{
			LabelSelector: "test=" + c.TestName,
		})
		if err != nil {
			return corev1.Pod{}, err
		} else if len(pods.Items) > 0 {
			for _, pod := range pods.Items {
				if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
					return pod, nil
				} else if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
					return pod, nil
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// streamLogs streams the logs from the given pod to stdout
func (c *TestController) streamLogs(pod corev1.Pod) error {
	req := c.kubeclient.CoreV1().Pods(c.TestName).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow: true,
	})
	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	defer readCloser.Close()

	buf := make([]byte, 1024)
	for {
		n, err := readCloser.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Print(string(buf[:n]))
	}
	return nil
}

// getStatus gets the status message and exit code of the given pod
func (c *TestController) getStatus(pod corev1.Pod) (string, int, error) {
	for {
		obj, err := c.kubeclient.CoreV1().Pods(c.TestName).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return "", 0, err
		} else {
			state := obj.Status.ContainerStatuses[0].State
			if state.Terminated != nil {
				return state.Terminated.Message, int(state.Terminated.ExitCode), nil
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// deleteClusterRoleBinding deletes the ClusterRoleBinding used by the test
func (c *TestController) deleteClusterRoleBinding() error {
	return c.kubeclient.RbacV1().ClusterRoleBindings().Delete("atomix-controller", &metav1.DeleteOptions{})
}

// deleteNamespace deletes the Namespace used by the test and all resources within it
func (c *TestController) deleteNamespace() error {
	return c.kubeclient.CoreV1().Namespaces().Delete(c.TestName, &metav1.DeleteOptions{})
}

// getTestName returns a qualified test name derived from the given test ID suitable for use in k8s resource names
func getTestName(testId string) string {
	return "onos-test-" + testId
}

// newKubeClient returns a new Kubernetes client from the environment
func newKubeClient() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		if home == "" {
			return nil, errors.New("no home directory configured")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	return kubernetes.NewForConfig(config)
}

// newExtensionsKubeClient returns a new extensions API server Kubernetes client from the environment
func newExtensionsKubeClient() (*apiextension.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		if home == "" {
			return nil, errors.New("no home directory configured")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	return apiextension.NewForConfig(config)
}

// newAtomixKubeClient returns a new Atomix Kubernetes client from the environment
func newAtomixKubeClient() (*atomixk8s.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		if home == "" {
			return nil, errors.New("no home directory configured")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	return atomixk8s.NewForConfig(config)
}

// homeDir returns the user's home directory if defined by environment variables
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
