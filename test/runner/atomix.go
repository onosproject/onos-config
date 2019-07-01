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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// setupAtomixController sets up the Atomix controller and associated resources
func (c *ClusterController) setupAtomixController() error {
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
			Namespace: c.clusterID,
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
			Name:      c.clusterID,
			Namespace: c.clusterID,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "atomix-controller",
				Namespace: c.clusterID,
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
			c.deleteAtomixClusterRoleBinding()
			return c.createAtomixClusterRoleBinding()
		}
		return err
	}
	return nil
}

// deleteAtomixClusterRoleBinding deletes the ClusterRoleBinding required by the Atomix controller
func (c *ClusterController) deleteAtomixClusterRoleBinding() error {
	return c.kubeclient.RbacV1().ClusterRoleBindings().Delete("atomix-controller", &metav1.DeleteOptions{})
}

// createAtomixServiceAccount creates a ServiceAccount used by the Atomix controller
func (c *ClusterController) createAtomixServiceAccount() error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.clusterID,
		},
	}
	_, err := c.kubeclient.CoreV1().ServiceAccounts(c.clusterID).Create(serviceAccount)
	return err
}

// createAtomixDeployment creates the Atomix controller Deployment
func (c *ClusterController) createAtomixDeployment() error {
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.clusterID,
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
	_, err := c.kubeclient.AppsV1().Deployments(c.clusterID).Create(deployment)
	return err
}

// createAtomixService creates a service for the controller
func (c *ClusterController) createAtomixService() error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.clusterID,
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
	_, err := c.kubeclient.CoreV1().Services(c.clusterID).Create(service)
	return err
}

// awaitAtomixControllerReady blocks until the Atomix controller is ready
func (c *ClusterController) awaitAtomixControllerReady() error {
	for {
		dep, err := c.kubeclient.AppsV1().Deployments(c.clusterID).Get("atomix-controller", metav1.GetOptions{})
		if err != nil {
			return err
		} else if dep.Status.ReadyReplicas == 1 {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
