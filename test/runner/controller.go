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
	"bufio"
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
	return getClusterName(c.ClusterId)
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
func (c *ClusterController) RunTests(tests []string, timeout time.Duration) (string, int, error) {
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

// setupAtomixController sets up the Atomix controller and associated resources
func (c *ClusterController) setupAtomixController() error {
	log.Infof("Setting up Atomix controller atomix-controller/%s", c.getClusterName())
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

	log.Infof("Waiting for Atomix controller atomix-controller/%s to become ready", c.getClusterName())
	if err := c.awaitAtomixControllerReady(); err != nil {
		return err
	}
	return nil
}

// createAtomixPartitionSetResource creates the PartitionSet custom resource definition in the k8s cluster
func (c *ClusterController) createAtomixPartitionSetResource() error {
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
func (c *ClusterController) createAtomixPartitionResource() error {
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
func (c *ClusterController) createAtomixClusterRole() error {
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.getClusterName(),
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
func (c *ClusterController) createAtomixClusterRoleBinding() error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.getClusterName(),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "atomix-controller",
				Namespace: c.getClusterName(),
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
func (c *ClusterController) createAtomixServiceAccount() error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.getClusterName(),
		},
	}
	_, err := c.kubeclient.CoreV1().ServiceAccounts(c.getClusterName()).Create(serviceAccount)
	return err
}

// createAtomixDeployment creates the Atomix controller Deployment
func (c *ClusterController) createAtomixDeployment() error {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.getClusterName(),
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
	_, err := c.kubeclient.AppsV1().Deployments(c.getClusterName()).Create(deployment)
	return err
}

// createAtomixService creates a service for the controller
func (c *ClusterController) createAtomixService() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.getClusterName(),
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
	_, err := c.kubeclient.CoreV1().Services(c.getClusterName()).Create(service)
	return err
}

// awaitAtomixControllerReady blocks until the Atomix controller is ready
func (c *ClusterController) awaitAtomixControllerReady() error {
	for {
		dep, err := c.kubeclient.AppsV1().Deployments(c.getClusterName()).Get("atomix-controller", metav1.GetOptions{})
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
func (c *ClusterController) setupPartitions() error {
	log.Infof("Setting up partitions raft/%s", c.getClusterName())
	if err := c.createPartitionSet(); err != nil {
		return err
	}

	log.Infof("Waiting for partitions raft/%s to become ready", c.getClusterName())
	if err := c.awaitPartitionsReady(); err != nil {
		return err
	}
	return nil
}

// createPartitionSet creates a Raft partition set from the configuration
func (c *ClusterController) createPartitionSet() error {
	bytes, err := yaml.Marshal(&raft.RaftProtocol{})
	if err != nil {
		return err
	}

	set := &v1alpha1.PartitionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "raft",
			Namespace: c.getClusterName(),
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
	_, err = c.atomixclient.K8sV1alpha1().PartitionSets(c.getClusterName()).Create(set)
	return err
}

// awaitPartitionsReady waits for Raft partitions to complete startup
func (c *ClusterController) awaitPartitionsReady() error {
	for {
		set, err := c.atomixclient.K8sV1alpha1().PartitionSets(c.getClusterName()).Get("raft", metav1.GetOptions{})
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
func (c *ClusterController) getDeviceIds() []string {
	devices := []string{}
	for name, _ := range c.config.DeviceStore["Store"].(map[string]interface{}) {
		devices = append(devices, name)
	}
	for name, _ := range c.config.Simulators {
		devices = append(devices, name)
	}
	return devices
}

// setupSimulator creates a simulator required for the test
func (c *ClusterController) setupSimulator(name string, config *SimulatorConfig) error {
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
func (c *ClusterController) createSimulatorConfigMap(name string, config *SimulatorConfig) error {
	configJson, err := json.Marshal(config)
	if err != nil {
		return err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.getClusterName(),
		},
		Data: map[string]string{
			"config.json": string(configJson),
		},
	}
	_, err = c.kubeclient.CoreV1().ConfigMaps(c.getClusterName()).Create(cm)
	return err
}

// createSimulatorPod creates a simulator pod
func (c *ClusterController) createSimulatorPod(name string) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.getClusterName(),
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
	_, err := c.kubeclient.CoreV1().Pods(c.getClusterName()).Create(pod)
	return err
}

// createSimulatorService creates a simulator service
func (c *ClusterController) createSimulatorService(name string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.getClusterName(),
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
	_, err := c.kubeclient.CoreV1().Services(c.getClusterName()).Create(service)
	return err
}

// awaitSimulatorReady waits for the given simulator to complete startup
func (c *ClusterController) awaitSimulatorReady(name string) error {
	for {
		pod, err := c.kubeclient.CoreV1().Pods(c.getClusterName()).Get(name, metav1.GetOptions{})
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
func (c *ClusterController) teardownSimulator(name string) error {
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
func (c *ClusterController) deleteSimulatorConfigMap(name string) error {
	return c.kubeclient.CoreV1().ConfigMaps(c.getClusterName()).Delete(name, &metav1.DeleteOptions{})
}

// deleteSimulatorPod deletes a simulator Pod by name
func (c *ClusterController) deleteSimulatorPod(name string) error {
	return c.kubeclient.CoreV1().Pods(c.getClusterName()).Delete(name, &metav1.DeleteOptions{})
}

// deleteSimulatorService deletes a simulator Service by name
func (c *ClusterController) deleteSimulatorService(name string) error {
	return c.kubeclient.CoreV1().Services(c.getClusterName()).Delete(name, &metav1.DeleteOptions{})
}

// setupOnosConfig sets up the onos-config Deployment
func (c *ClusterController) setupOnosConfig() error {
	log.Infof("Setting up onos-config cluster onos-config/%s", c.getClusterName())
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

	log.Infof("Waiting for onos-config cluster onos-config/%s to become ready", c.getClusterName())
	if err := c.awaitOnosConfigDeploymentReady(); err != nil {
		return err
	}
	return nil
}

// createOnosConfigSecret creates a secret for configuring TLS in onos-config and clients
func (c *ClusterController) createOnosConfigSecret() error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.getClusterName(),
			Namespace: c.getClusterName(),
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

	_, err = c.kubeclient.CoreV1().Secrets(c.getClusterName()).Create(secret)
	return err
}

// createOnosConfigConfigMap creates a ConfigMap for the onos-config Deployment
func (c *ClusterController) createOnosConfigConfigMap() error {
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
	deviceStoreObj := c.config.DeviceStore
	configStoreObj := c.config.ConfigStore
	deviceStoreMap := deviceStoreObj["Store"].(map[string]interface{})
	configStoreMap := configStoreObj["Store"].(map[string]interface{})
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
	deviceStore, err := json.Marshal(deviceStoreObj)
	if err != nil {
		return err
	}

	// Serialize the config store configuration
	configStore, err := json.Marshal(configStoreObj)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.getClusterName(),
		},
		Data: map[string]string{
			"changeStore.json":  string(changeStore),
			"configStore.json":  string(configStore),
			"deviceStore.json":  string(deviceStore),
			"networkStore.json": string(networkStore),
		},
	}
	_, err = c.kubeclient.CoreV1().ConfigMaps(c.getClusterName()).Create(cm)
	return err
}

// createOnosConfigDeployment creates an onos-config Deployment
func (c *ClusterController) createOnosConfigDeployment() error {
	nodes := int32(c.config.Nodes)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.getClusterName(),
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
									Value: fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", c.getClusterName()),
								},
								{
									Name:  "ATOMIX_APP",
									Value: "test",
								},
								{
									Name:  "ATOMIX_NAMESPACE",
									Value: c.getClusterName(),
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
									SecretName: c.getClusterName(),
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := c.kubeclient.AppsV1().Deployments(c.getClusterName()).Create(dep)
	return err
}

// createOnosConfigService creates a Service to expose the onos-config Deployment to other pods
func (c *ClusterController) createOnosConfigService() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.getClusterName(),
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
	_, err := c.kubeclient.CoreV1().Services(c.getClusterName()).Create(service)
	return err
}

// awaitOnosConfigDeploymentReady waits for the onos-config pods to complete startup
func (c *ClusterController) awaitOnosConfigDeploymentReady() error {
	for {
		dep, err := c.kubeclient.AppsV1().Deployments(c.getClusterName()).Get("onos-config", metav1.GetOptions{})
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
func (c *ClusterController) redeployOnosConfig() error {
	log.Infof("Redeploying onos-config cluster onos-config/%s", c.getClusterName())
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
	log.Infof("Waiting for onos-config cluster onos-config/%s to become ready", c.getClusterName())
	if err := c.awaitOnosConfigDeploymentReady(); err != nil {
		return err
	}
	return nil
}

// deleteOnosConfigConfigMap deletes the onos-config ConfigMap
func (c *ClusterController) deleteOnosConfigConfigMap() error {
	return c.kubeclient.CoreV1().ConfigMaps(c.getClusterName()).Delete("onos-config", &metav1.DeleteOptions{})
}

// deleteOnosConfigDeployment deletes the onos-config Deployment
func (c *ClusterController) deleteOnosConfigDeployment() error {
	return c.kubeclient.AppsV1().Deployments(c.getClusterName()).Delete("onos-config", &metav1.DeleteOptions{})
}

// start starts running the test job
func (c *ClusterController) start(args []string, timeout time.Duration) (corev1.Pod, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return corev1.Pod{}, err
	}

	testId := id.String()
	if err := c.createTestJob(testId, args, timeout); err != nil {
		return corev1.Pod{}, err
	}
	return c.awaitTestJobRunning(testId)
}

// createTestJob creates the job to run tests
func (c *ClusterController) createTestJob(testId string, args []string, timeout time.Duration) error {
	log.Infof("Starting test job %s", getTestName(testId))
	one := int32(1)
	timeoutSeconds := int64(timeout / time.Second)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getTestName(testId),
			Namespace: c.getClusterName(),
			Annotations: map[string]string{
				"test-args": strings.Join(args, ","),
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:           &one,
			Completions:           &one,
			BackoffLimit:          &one,
			ActiveDeadlineSeconds: &timeoutSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster": c.ClusterId,
						"test":    testId,
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
									SecretName: c.getClusterName(),
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.kubeclient.BatchV1().Jobs(c.getClusterName()).Create(job)
	return err
}

// awaitTestJobRunning blocks until the test job creates a pod in the RUNNING state
func (c *ClusterController) awaitTestJobRunning(testId string) (corev1.Pod, error) {
	log.Infof("Waiting for test job %s to become ready", testId)
	for {
		pod, err := c.getPod(testId)
		if err == nil {
			return pod, nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// streamLogs streams the logs from the given pod to stdout
func (c *ClusterController) streamLogs(pod corev1.Pod) error {
	req := c.kubeclient.CoreV1().Pods(c.getClusterName()).GetLogs(pod.Name, &corev1.PodLogOptions{
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
func (c *ClusterController) getStatus(pod corev1.Pod) (string, int, error) {
	for {
		obj, err := c.kubeclient.CoreV1().Pods(c.getClusterName()).Get(pod.Name, metav1.GetOptions{})
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
func (c *ClusterController) deleteClusterRoleBinding() error {
	return c.kubeclient.RbacV1().ClusterRoleBindings().Delete("atomix-controller", &metav1.DeleteOptions{})
}

// deleteNamespace deletes the Namespace used by the test and all resources within it
func (c *ClusterController) deleteNamespace() error {
	return c.kubeclient.CoreV1().Namespaces().Delete(c.getClusterName(), &metav1.DeleteOptions{})
}

// GetHistory returns the history of test runs on the cluster
func (c *ClusterController) GetHistory() ([]TestRecord, error) {
	jobs, err := c.kubeclient.BatchV1().Jobs(c.getClusterName()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	records := make([]TestRecord, 0, len(jobs.Items))
	for _, job := range jobs.Items {
		record, err := c.getRecord(job)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// GetRecord returns a single record for the given test
func (c *ClusterController) GetRecord(testId string) (TestRecord, error) {
	job, err := c.kubeclient.BatchV1().Jobs(c.getClusterName()).Get(getTestName(testId), metav1.GetOptions{})
	if err != nil {
		return TestRecord{}, err
	}
	return c.getRecord(*job)
}

// GetRecord returns a single record for the given test
func (c *ClusterController) getRecord(job batchv1.Job) (TestRecord, error) {
	testId := job.Labels["test"]

	var args []string
	testArgs, ok := job.Annotations["test-args"]
	if ok {
		args = strings.Split(testArgs, ",")
	} else {
		args = make([]string, 0)
	}

	pod, err := c.getPod(testId)
	if err != nil {
		return TestRecord{}, nil
	}

	logs, err := c.getLogs(pod)
	if err != nil {
		return TestRecord{}, nil
	}

	record := TestRecord{
		TestId: testId,
		Args:   args,
		Logs:   logs,
	}

	state := pod.Status.ContainerStatuses[0].State
	if state.Terminated != nil {
		record.Message = state.Terminated.Message
		record.ExitCode = int(state.Terminated.ExitCode)
		if record.ExitCode == 0 {
			record.Status = TestPassed
		} else {
			record.Status = TestFailed
		}
	} else {
		record.Status = TestRunning
	}

	return record, nil
}

// getLogs gets the logs from the given pod
func (c *ClusterController) getLogs(pod corev1.Pod) ([]string, error) {
	req := c.kubeclient.CoreV1().Pods(c.getClusterName()).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow: true,
	})
	readCloser, err := req.Stream()
	if err != nil {
		return nil, err
	}

	defer readCloser.Close()

	logs := []string{}
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}
	return logs, nil
}

// getPod finds the Pod for the given test
func (c *ClusterController) getPod(testId string) (corev1.Pod, error) {
	pods, err := c.kubeclient.CoreV1().Pods(c.getClusterName()).List(metav1.ListOptions{
		LabelSelector: "test=" + testId,
	})
	if err != nil {
		return corev1.Pod{}, err
	} else if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
				return pod, nil
			}
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				return pod, nil
			}
		}
	}
	return corev1.Pod{}, errors.New("cannot locate test pod for test " + testId)
}

// TestRecord contains information about a test run
type TestRecord struct {
	TestId   string
	Args     []string
	Logs     []string
	Status   TestStatus
	Message  string
	ExitCode int
}

type TestStatus string

const (
	TestRunning TestStatus = "RUNNING"
	TestPassed  TestStatus = "PASSED"
	TestFailed  TestStatus = "FAILED"
)

// getClusterName returns a qualified cluster name derived from the given cluster ID
func getClusterName(clusterId string) string {
	return fmt.Sprintf("onos-cluster-%s", clusterId)
}

// getTestName returns a qualified test name derived from the given test ID suitable for use in k8s resource names
func getTestName(testId string) string {
	return fmt.Sprintf("onos-test-%s", testId)
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
