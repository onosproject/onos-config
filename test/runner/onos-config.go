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
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"gopkg.in/yaml.v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// setupOnosConfig sets up the onos-config Deployment
func (c *ClusterController) setupOnosConfig() error {
	if err := c.createOnosConfigConfigMap(); err != nil {
		return err
	}
	if err := c.createOnosConfigService(); err != nil {
		return err
	}
	if err := c.createOnosConfigDeployment(); err != nil {
		return err
	}
	if err := c.awaitOnosConfigDeploymentReady(); err != nil {
		return err
	}
	return nil
}

// createOnosConfigConfigMap creates a ConfigMap for the onos-config Deployment
func (c *ClusterController) createOnosConfigConfigMap() error {
	config, err := c.config.load()
	if err != nil {
		return err
	}

	// Serialize the change store configuration
	changeStore, err := json.Marshal(config["changeStore"])
	if err != nil {
		return err
	}

	// Serialize the network store configuration
	networkStore, err := json.Marshal(config["networkStore"])
	if err != nil {
		return err
	}

	// Serialize the device store configuration
	deviceStore, err := json.Marshal(config["deviceStore"])
	if err != nil {
		return err
	}

	// Serialize the config store configuration
	configStore, err := json.Marshal(config["configStore"])
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.clusterID,
		},
		Data: map[string]string{
			"changeStore.json":  string(changeStore),
			"configStore.json":  string(configStore),
			"deviceStore.json":  string(deviceStore),
			"networkStore.json": string(networkStore),
		},
	}
	_, err = c.kubeclient.CoreV1().ConfigMaps(c.clusterID).Create(cm)
	return err
}

// addSimulatorToConfig adds a simulator to the onos-config configuration
func (c *ClusterController) addSimulatorToConfig(name string) error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "config"}}
	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if err = c.addSimulatorToPod(name, pod); err != nil {
			return err
		}
	}
	return nil
}

// addNetworkToConfig adds a network to the onos-config configuration
func (c *ClusterController) addNetworkToConfig(name string, config *NetworkConfig) error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "config"}}
	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		var port = 50001
		for i := 0; i < config.NumDevices; i++ {
			var buf bytes.Buffer
			buf.WriteString(name)
			buf.WriteString("-s")
			buf.WriteString(strconv.Itoa(i))
			deviceName := buf.String()
			if err = c.addNetworkToPod(deviceName, port, pod); err != nil {
				return err
			}
			port = port + 1
		}
	}
	return nil
}

// addSimulatorToPod adds the given simulator to the given pod's configuration
func (c *ClusterController) addSimulatorToPod(name string, pod corev1.Pod) error {
	command := fmt.Sprintf("onos devices add \"id: '%s', address: '%s:10161' version: '1.0.0' devicetype: 'Devicesim'\" --address 127.0.0.1:5150 --keyPath /etc/onos-config/certs/client1.key --certPath /etc/onos-config/certs/client1.crt", name, name)
	return c.execute(pod, []string{"/bin/bash", "-c", command})
}

// addNetworkToPod adds the given network to the given pod's configuration
func (c *ClusterController) addNetworkToPod(name string, port int, pod corev1.Pod) error {
	command := fmt.Sprintf("onos devices add \"id: '%s', address: '%s:%s' version: '1.0.0' devicetype: 'Stratum'\" --address 127.0.0.1:5150 --keyPath /etc/onos-config/certs/client1.key --certPath /etc/onos-config/certs/client1.crt", name, name, strconv.Itoa(port))
	return c.execute(pod, []string{"/bin/bash", "-c", command})
}

// removeSimulatorFromConfig removes a simulator from the onos-config configuration
func (c *ClusterController) removeSimulatorFromConfig(name string) error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "config"}}
	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		if err = c.removeSimulatorFromPod(name, pod); err != nil {
			return err
		}
	}
	return nil
}

// removeNetworkFromConfig removes a network from the onos-config configuration
func (c *ClusterController) removeNetworkFromConfig(name string, configMap *corev1.ConfigMapList) error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "config"}}
	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	})
	if err != nil {
		return err
	}
	dataMap := configMap.Items[0].BinaryData["config"]
	m := make(map[string]interface{})
	_ = yaml.Unmarshal(dataMap, &m)
	numDevices := m["numdevices"].(int)

	for _, pod := range pods.Items {
		for i := 0; i < numDevices; i++ {
			var buf bytes.Buffer
			buf.WriteString(name)
			buf.WriteString("-s")
			buf.WriteString(strconv.Itoa(i))
			deviceName := buf.String()
			if err = c.removeNetworkFromPod(deviceName, pod); err != nil {
				return err
			}
		}
	}
	return nil
}

// removeSimulatorFromPod removes the given simulator from the given pod
func (c *ClusterController) removeSimulatorFromPod(name string, pod corev1.Pod) error {
	command := fmt.Sprintf("onos devices remove %s --address 127.0.0.1:5150 --keyPath /etc/onos-config/certs/client1.key --certPath /etc/onos-config/certs/client1.crt", name)
	return c.execute(pod, []string{"/bin/bash", "-c", command})
}

// removeNetworkFromPod removes the given network from the given pod
func (c *ClusterController) removeNetworkFromPod(name string, pod corev1.Pod) error {
	command := fmt.Sprintf("onos devices remove %s --address 127.0.0.1:5150 --keyPath /etc/onos-config/certs/client1.key --certPath /etc/onos-config/certs/client1.crt", name)
	return c.execute(pod, []string{"/bin/bash", "-c", command})
}

// createOnosConfigDeployment creates an onos-config Deployment
func (c *ClusterController) createOnosConfigDeployment() error {
	nodes := int32(c.config.ConfigNodes)
	zero := int64(0)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.clusterID,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &nodes,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  "onos",
					"type": "config",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "onos",
						"type":     "config",
						"resource": "onos-config",
					},
					Annotations: map[string]string{
						"seccomp.security.alpha.kubernetes.io/pod": "unconfined",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "onos-config",
							Image:           "onosproject/onos-config:debug",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env: []corev1.EnvVar{
								{
									Name:  "ATOMIX_CONTROLLER",
									Value: fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", c.clusterID),
								},
								{
									Name:  "ATOMIX_APP",
									Value: "test",
								},
								{
									Name:  "ATOMIX_NAMESPACE",
									Value: c.clusterID,
								},
							},
							Args: []string{
								"-caPath=/etc/onos-config/certs/onf.cacrt",
								"-keyPath=/etc/onos-config/certs/onos-config.key",
								"-certPath=/etc/onos-config/certs/onos-config.crt",
								"-configStore=/etc/onos-config/configs/configStore.json",
								"-changeStore=/etc/onos-config/configs/changeStore.json",
								"-deviceStore=/etc/onos-config/configs/deviceStore.json",
								"-networkStore=/etc/onos-config/configs/networkStore.json",
								"-modelPlugin=/usr/local/lib/testdevice-debug.so.1.0.0",
								"-modelPlugin=/usr/local/lib/testdevice-debug.so.2.0.0",
								"-modelPlugin=/usr/local/lib/devicesim-debug.so.1.0.0",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpc",
									ContainerPort: 5150,
								},
								{
									Name:          "debug",
									ContainerPort: 40000,
									Protocol:      corev1.ProtocolTCP,
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
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_PTRACE",
									},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &zero,
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
									SecretName: c.clusterID,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := c.kubeclient.AppsV1().Deployments(c.clusterID).Create(dep)
	return err
}

// createOnosConfigService creates a Service to expose the onos-config Deployment to other pods
func (c *ClusterController) createOnosConfigService() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-config",
			Namespace: c.clusterID,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":  "onos",
				"type": "config",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "grpc",
					Port: 5150,
				},
			},
		},
	}
	_, err := c.kubeclient.CoreV1().Services(c.clusterID).Create(service)
	return err
}

// awaitOnosConfigDeploymentReady waits for the onos-config pods to complete startup
func (c *ClusterController) awaitOnosConfigDeploymentReady() error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "config"}}
	unblocked := make(map[string]bool)
	for {
		// Get a list of the pods that match the deployment
		pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		})
		if err != nil {
			return err
		}

		// Iterate through the pods in the deployment and unblock the debugger
		for _, pod := range pods.Items {
			if _, ok := unblocked[pod.Name]; !ok && len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].State.Running != nil {
				err := c.execute(pod, []string{"/bin/bash", "-c", "dlv --init <(echo \"exit -c\") connect 127.0.0.1:40000"})
				if err != nil {
					return err
				}
				unblocked[pod.Name] = true
			}
		}

		// Get the onos-config deployment
		dep, err := c.kubeclient.AppsV1().Deployments(c.clusterID).Get("onos-config", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Return once the all replicas in the deployment are ready
		if int(dep.Status.ReadyReplicas) == c.config.ConfigNodes {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// GetOnosConfigNodes returns a list of all onos-config nodes running in the cluster
func (c *ClusterController) GetOnosConfigNodes() ([]NodeInfo, error) {
	configLabelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "config"}}

	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: labels.Set(configLabelSelector.MatchLabels).String(),
	})
	if err != nil {
		return nil, err
	}

	onosConfigNodes := make([]NodeInfo, len(pods.Items))
	for i, pod := range pods.Items {
		var status NodeStatus
		if pod.Status.Phase == corev1.PodRunning {
			status = NodeRunning
		} else if pod.Status.Phase == corev1.PodFailed {
			status = NodeFailed
		}
		onosConfigNodes[i] = NodeInfo{
			ID:     pod.Name,
			Status: status,
			Type:   OnosConfig,
		}
	}

	return onosConfigNodes, nil
}
