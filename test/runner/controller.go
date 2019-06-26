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
	atomixk8s "github.com/atomix/atomix-k8s-controller/pkg/client/clientset/versioned"
	"gopkg.in/yaml.v1"
	corev1 "k8s.io/api/core/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NewController creates a new onit controller
func NewController() (*OnitController, error) {
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

	return &OnitController{
		kubeclient:       kubeclient,
		atomixclient:     atomixclient,
		extensionsclient: extensionsclient,
	}, nil
}

// OnitController manages clusters for onit
type OnitController struct {
	kubeclient       *kubernetes.Clientset
	atomixclient     *atomixk8s.Clientset
	extensionsclient *apiextension.Clientset
}

// GetClusters returns a list of onit clusters
func (c *OnitController) GetClusters() (map[string]*ClusterConfig, error) {
	namespaces, err := c.kubeclient.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: "app=onit",
	})
	if err != nil {
		return nil, err
	}

	clusters := make(map[string]*ClusterConfig)
	for _, ns := range namespaces.Items {
		name := ns.Name
		cm, err := c.kubeclient.CoreV1().ConfigMaps(name).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		config := &ClusterConfig{}
		if err = yaml.Unmarshal(cm.BinaryData["config"], config); err != nil {
			return nil, err
		}
		clusters[name] = config
	}
	return clusters, nil
}

// NewCluster creates a new cluster controller
func (c *OnitController) NewCluster(clusterID string, config *ClusterConfig) (*ClusterController, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterID,
			Labels: map[string]string{
				"app": "onit",
			},
		},
	}
	_, err := c.kubeclient.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return nil, err
	}

	configString, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterID,
			Namespace: clusterID,
		},
		BinaryData: map[string][]byte{
			"config": configString,
		},
	}
	_, err = c.kubeclient.CoreV1().ConfigMaps(clusterID).Create(cm)
	if err != nil {
		return nil, err
	}

	return &ClusterController{
		clusterID:        clusterID,
		kubeclient:       c.kubeclient,
		atomixclient:     c.atomixclient,
		extensionsclient: c.extensionsclient,
		config:           config,
	}, nil
}

// GetCluster returns a cluster controller
func (c *OnitController) GetCluster(clusterID string) (*ClusterController, error) {
	_, err := c.kubeclient.CoreV1().Namespaces().Get(clusterID, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cm, err := c.kubeclient.CoreV1().ConfigMaps(clusterID).Get(clusterID, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	config := &ClusterConfig{}
	if err = yaml.Unmarshal(cm.BinaryData["config"], config); err != nil {
		return nil, err
	}

	return &ClusterController{
		clusterID:        clusterID,
		kubeclient:       c.kubeclient,
		atomixclient:     c.atomixclient,
		extensionsclient: c.extensionsclient,
		config:           config,
	}, nil
}

// DeleteCluster deletes a cluster controller
func (c *OnitController) DeleteCluster(clusterID string) error {
	if err := c.kubeclient.RbacV1().ClusterRoleBindings().Delete("atomix-controller", &metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := c.kubeclient.CoreV1().Namespaces().Delete(clusterID, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}
