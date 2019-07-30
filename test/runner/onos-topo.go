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
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// setupOnosTopo sets up the onos-topo Deployment
func (c *ClusterController) setupOnosTopo() error {
	if err := c.createOnosTopoConfigMap(); err != nil {
		return err
	}

	if err := c.createOnosTopoService(); err != nil {
		return err
	}

	if err := c.createOnosTopoDeployment(); err != nil {
		return err
	}
	if err := c.awaitOnosTopoDeploymentReady(); err != nil {
		return err
	}

	return nil
}

// createOnosTopoConfigMap creates a ConfigMap for the onos-topo Deployment
func (c *ClusterController) createOnosTopoConfigMap() error {

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-topo",
			Namespace: c.clusterID,
		},
		Data: map[string]string{},
	}
	_, err := c.kubeclient.CoreV1().ConfigMaps(c.clusterID).Create(cm)
	return err
}

// createOnosTopoService creates a Service to expose the onos-topo Deployment to other pods
func (c *ClusterController) createOnosTopoService() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-topo",
			Namespace: c.clusterID,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":  "onos",
				"type": "topo",
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

// createOnosTopoDeployment creates an onos-topo Deployment
func (c *ClusterController) createOnosTopoDeployment() error {
	nodes := int32(c.config.TopoNodes)
	zero := int64(0)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "onos-topo",
			Namespace: c.clusterID,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &nodes,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  "onos",
					"type": "topo",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "onos",
						"type":     "topo",
						"resource": "onos-topo",
					},
					Annotations: map[string]string{
						"seccomp.security.alpha.kubernetes.io/pod": "unconfined",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "onos-topo",
							Image:           "onosproject/onos-topo:debug",
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
								"-caPath=/etc/onos-topo/certs/onf.cacrt",
								"-keyPath=/etc/onos-topo/certs/onos-config.key",
								"-certPath=/etc/onos-topo/certs/onos-config.crt",
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
									Name:      "topo",
									MountPath: "/etc/onos-topo/configs",
									ReadOnly:  true,
								},
								{
									Name:      "secret",
									MountPath: "/etc/onos-topo/certs",
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
							Name: "topo",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "onos-topo",
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

// awaitOnosTopoDeploymentReady waits for the onos-topo pods to complete startup
func (c *ClusterController) awaitOnosTopoDeploymentReady() error {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "topo"}}
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

		// Get the onos-topo deployment
		dep, err := c.kubeclient.AppsV1().Deployments(c.clusterID).Get("onos-topo", metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Return once the all replicas in the deployment are ready
		if int(dep.Status.ReadyReplicas) == c.config.TopoNodes {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// GetOnosTopoNodes returns a list of all onos-topo nodes running in the cluster
func (c *ClusterController) GetOnosTopoNodes() ([]NodeInfo, error) {
	topoLabelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": "onos", "type": "topo"}}

	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: labels.Set(topoLabelSelector.MatchLabels).String(),
	})
	if err != nil {
		return nil, err
	}

	onosTopoNodes := make([]NodeInfo, len(pods.Items))
	for i, pod := range pods.Items {
		var status NodeStatus
		if pod.Status.Phase == corev1.PodRunning {
			status = NodeRunning
		} else if pod.Status.Phase == corev1.PodFailed {
			status = NodeFailed
		}
		onosTopoNodes[i] = NodeInfo{
			ID:     pod.Name,
			Status: status,
			Type:   OnosTopo,
		}
	}

	return onosTopoNodes, nil
}
