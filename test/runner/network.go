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
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	topoOption = "--topo"
)

// TopoType topology type
type TopoType int

const (
	// Linear linear topology type
	Linear TopoType = iota
	// Tree tree topology type
	Tree
	// Single node topology
	Single
)

func (d TopoType) String() string {
	return [...]string{"linear", "tree", "single"}[d]
}

// GetNetworks returns a list of networks deployed in the cluster
func (c *ClusterController) GetNetworks() ([]string, error) {
	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: "type=network",
	})

	if err != nil {
		return nil, err
	}

	networks := make([]string, len(pods.Items))
	for i, pod := range pods.Items {
		networks[i] = pod.Name
	}
	return networks, nil
}

// setupNetwork creates a network of stratum devices required for the test
func (c *ClusterController) setupNetwork(name string, config *NetworkConfig) error {

	if err := c.createNetworkConfigMap(name, config); err != nil {
		return err
	}
	if err := c.createNetworkPod(name, config); err != nil {
		return err
	}
	if err := c.createNetworkService(name, config); err != nil {
		return err
	}
	if err := c.awaitNetworkReady(name); err != nil {
		return err
	}
	return nil
}

// createNetworkConfigMap creates a network configuration
func (c *ClusterController) createNetworkConfigMap(name string, config *NetworkConfig) error {

	configByte, err := yaml.Marshal(config)

	if err != nil {
		return err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.clusterID,
			Labels: map[string]string{
				"network": name,
			},
		},

		BinaryData: map[string][]byte{
			"config": configByte,
		},
	}
	_, err = c.kubeclient.CoreV1().ConfigMaps(c.clusterID).Create(cm)
	if err != nil {
		return err
	}

	return err
}

// createNetworkPod creates a stratum network pod
func (c *ClusterController) createNetworkPod(name string, config *NetworkConfig) error {

	var isPrivileged = true
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.clusterID,
			Labels: map[string]string{
				"type":    "network",
				"network": name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "stratum-simulator",
					Image:           "opennetworking/mn-stratum",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Stdin:           true,
					Args:            config.MininetOptions,
					Ports: []corev1.ContainerPort{
						{
							Name:          "stratum",
							ContainerPort: 50001,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt(50001),
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
					},
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							TCPSocket: &corev1.TCPSocketAction{
								Port: intstr.FromInt(50001),
							},
						},
						InitialDelaySeconds: 15,
						PeriodSeconds:       20,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: &isPrivileged,
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

	_, err := c.kubeclient.CoreV1().Pods(c.clusterID).Create(pod)
	return err
}

// awaitSimulatorReady waits for the given simulator to complete startup
func (c *ClusterController) awaitNetworkReady(name string) error {
	for {
		pod, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		} else if len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// ParseMininetOptions parses mininet options and initialize the network configuration accordingly
func ParseMininetOptions(config *NetworkConfig) {
	mininetOptions := config.MininetOptions

	if len(mininetOptions) != 0 {
		if strings.Compare(mininetOptions[0], topoOption) == 0 {
			topoType := strings.Split(mininetOptions[1], ",")
			if strings.Compare(topoType[0], Linear.String()) == 0 {

				config.NumDevices, _ = strconv.Atoi(topoType[1])
				config.TopoType = Linear
			}
		}
	} else {
		config.NumDevices = 1
		config.TopoType = Single

	}
}

// createNetworkService creates a network service
func (c *ClusterController) createNetworkService(name string, config *NetworkConfig) error {

	var port int32 = 50001

	for i := 0; i < config.NumDevices; i++ {
		var buf bytes.Buffer
		buf.WriteString(name)
		buf.WriteString("-s")
		buf.WriteString(strconv.Itoa(i))
		serviceName := buf.String()

		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: c.clusterID,
				Labels:    map[string]string{"network": name},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"network": name,
				},
				Ports: []corev1.ServicePort{
					{
						Name: serviceName,
						Port: port,
					},
				},
			},
		}
		_, err := c.kubeclient.CoreV1().Services(c.clusterID).Create(service)
		if err != nil {
			return err
		}
		port = port + 1
	}

	return nil
}

// teardownNetwork tears down a network by name
func (c *ClusterController) teardownNetwork(name string) error {
	var err error
	if e := c.deleteNetworkPod(name); e != nil {
		err = e
	}
	if e := c.deleteNetworkService(name); e != nil {
		err = e
	}
	if e := c.deleteNetworkConfigMap(name); e != nil {
		err = e
	}
	return err
}

// deleteNetworkConfigMap deletes a network ConfigMap by name
func (c *ClusterController) deleteNetworkConfigMap(name string) error {
	return c.kubeclient.CoreV1().ConfigMaps(c.clusterID).Delete(name, &metav1.DeleteOptions{})
}

// deleteNetworkPod deletes a network Pod by name
func (c *ClusterController) deleteNetworkPod(name string) error {
	return c.kubeclient.CoreV1().Pods(c.clusterID).Delete(name, &metav1.DeleteOptions{})
}

// deleteNetworkService deletes all network Service by name
func (c *ClusterController) deleteNetworkService(name string) error {
	label := "network=" + name
	serviceList, _ := c.kubeclient.CoreV1().Services(c.clusterID).List(metav1.ListOptions{
		LabelSelector: label,
	})

	for _, svc := range serviceList.Items {
		err := c.kubeclient.CoreV1().Services(c.clusterID).Delete(svc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
