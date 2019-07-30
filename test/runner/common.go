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
	"io/ioutil"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// NodeStatus node status
type NodeStatus string

const (
	// NodeRunning node is running
	NodeRunning NodeStatus = "RUNNING"

	// NodeFailed node has failed
	NodeFailed NodeStatus = "FAILED"
)

// NodeType node type
type NodeType string

const (
	// OnosConfig  type of node is config
	OnosConfig NodeType = "config"

	// OnosTopo type of node is topo
	OnosTopo NodeType = "topo"

	// OnosAll type of node is all
	OnosAll NodeType = "all"
)

// NodeInfo contains information about a node
type NodeInfo struct {
	ID     string
	Status NodeStatus
	Type   NodeType
}

// GetNodes returns a list of all onos nodes  running in the cluster
func (c *ClusterController) GetNodes() ([]NodeInfo, error) {

	onosTopoNodes, _ := c.GetOnosTopoNodes()
	onosConfigNodes, _ := c.GetOnosConfigNodes()
	nodes := append(onosTopoNodes, onosConfigNodes...)

	return nodes, nil
}

// execute executes a command in the given pod
func (c *ClusterController) execute(pod corev1.Pod, command []string) error {
	container := pod.Spec.Containers[0]
	req := c.kubeclient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", container.Name)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: container.Name,
		Command:   command,
		Stdout:    true,
		Stderr:    true,
		Stdin:     false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.restconfig, "POST", req.URL())
	if err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		print(stdout.String())
		print(stderr.String())
	}
	return err

}

// createOnosSecret creates a secret for configuring TLS in onos nodes and clients
func (c *ClusterController) createOnosSecret() error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.clusterID,
			Namespace: c.clusterID,
		},
		StringData: map[string]string{},
	}

	err := filepath.Walk(certsPath, func(path string, info os.FileInfo, walkError error) error {
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

	_, err = c.kubeclient.CoreV1().Secrets(c.clusterID).Create(secret)
	return err
}
