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
	"github.com/onosproject/onos-config/test/console"
	"gopkg.in/yaml.v1"
	corev1 "k8s.io/api/core/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewController creates a new onit controller
func NewController() (*OnitController, error) {
	restconfig, err := getRestConfig()
	if err != nil {
		return nil, err
	}

	kubeclient, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	atomixclient, err := atomixk8s.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	extensionsclient, err := apiextension.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &OnitController{
		restconfig:       restconfig,
		kubeclient:       kubeclient,
		atomixclient:     atomixclient,
		extensionsclient: extensionsclient,
		status:           console.NewStatusWriter(),
	}, nil
}

// OnitController manages clusters for onit
type OnitController struct {
	restconfig       *rest.Config
	kubeclient       *kubernetes.Clientset
	atomixclient     *atomixk8s.Clientset
	extensionsclient *apiextension.Clientset
	status           *console.StatusWriter
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
		if ns.Status.Phase == corev1.NamespaceActive {
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
	}
	return clusters, nil
}

// NewCluster creates a new cluster controller
func (c *OnitController) NewCluster(clusterID string, config *ClusterConfig) (*ClusterController, console.ErrorStatus) {
	c.status.Start("Creating cluster namespace")
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
		return nil, c.status.Fail(err)
	}

	configString, err := yaml.Marshal(config)
	if err != nil {
		return nil, c.status.Fail(err)
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
		return nil, c.status.Fail(err)
	}

	return &ClusterController{
		clusterID:        clusterID,
		restconfig:       c.restconfig,
		kubeclient:       c.kubeclient,
		atomixclient:     c.atomixclient,
		extensionsclient: c.extensionsclient,
		config:           config,
		status:           c.status,
	}, c.status.Succeed()
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
		restconfig:       c.restconfig,
		kubeclient:       c.kubeclient,
		atomixclient:     c.atomixclient,
		extensionsclient: c.extensionsclient,
		config:           config,
		status:           c.status,
	}, nil
}

// DeleteCluster deletes a cluster controller
func (c *OnitController) DeleteCluster(clusterID string) console.ErrorStatus {
	c.status.Start("Deleting cluster namespace")
	if err := c.kubeclient.RbacV1().ClusterRoleBindings().Delete(clusterID, &metav1.DeleteOptions{}); err != nil {
		c.status.Fail(err)
	}
	if err := c.kubeclient.CoreV1().Namespaces().Delete(clusterID, &metav1.DeleteOptions{}); err != nil {
		return c.status.Fail(err)
	}
	return c.status.Succeed()
}
