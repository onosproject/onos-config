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
	"fmt"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	log "k8s.io/klog"
	"os"
	"path/filepath"
	"time"
)

type NodeStatus string

const (
	NodeRunning NodeStatus = "RUNNING"
	NodeFailed  NodeStatus = "FAILED"
)

// NodeInfo contains information about an onos-config node
type NodeInfo struct {
	Id     string
	Status NodeStatus
}

// GetNodes returns a list of onos-config nodes running in the cluster
func (c *ClusterController) GetNodes() ([]NodeInfo, error) {
	pods, err := c.kubeclient.CoreV1().Pods(c.getClusterName()).List(metav1.ListOptions{
		LabelSelector: "app=onos-config",
	})
	if err != nil {
		return nil, err
	}

	nodes := make([]NodeInfo, len(pods.Items))
	for i, pod := range pods.Items {
		var status NodeStatus
		if pod.Status.Phase == corev1.PodRunning {
			status = NodeRunning
		} else if pod.Status.Phase == corev1.PodFailed {
			status = NodeFailed
		}
		nodes[i] = NodeInfo{
			Id:     pod.Name,
			Status: status,
		}
	}
	return nodes, nil
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
						"resource": "onos-config",
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
