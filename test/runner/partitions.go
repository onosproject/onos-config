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
	"github.com/atomix/atomix-k8s-controller/pkg/apis/k8s/v1alpha1"
	raft "github.com/atomix/atomix-k8s-controller/proto/atomix/protocols/raft"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"time"
)

// PartitionInfo contains information about storage partitions
type PartitionInfo struct {
	Group     string
	Partition int
	Nodes     []string
}

// GetPartitions returns a list of partition info
func (c *ClusterController) GetPartitions() ([]PartitionInfo, error) {
	partitionList, err := c.atomixclient.K8sV1alpha1().Partitions(c.clusterID).List(metav1.ListOptions{
		LabelSelector: "group=raft",
	})
	if err != nil {
		return nil, err
	}

	partitions := make([]PartitionInfo, len(partitionList.Items))
	for i, partition := range partitionList.Items {
		partitionID, err := strconv.ParseInt(partition.Labels["partition"], 0, 32)
		if err != nil {
			return nil, err
		}

		nodes, err := c.GetPartitionNodes(int(partitionID))
		if err != nil {
			return nil, err
		}
		nodeIds := make([]string, len(nodes))
		for j, node := range nodes {
			nodeIds[j] = node.ID
		}

		partitions[i] = PartitionInfo{
			Group:     "raft",
			Partition: int(partitionID),
			Nodes:     nodeIds,
		}
	}
	return partitions, nil
}

// GetPartitionNodes returns a list of node info for the given partition
func (c *ClusterController) GetPartitionNodes(partition int) ([]NodeInfo, error) {
	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("group=raft,partition=%d", partition),
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
			ID:     pod.Name,
			Status: status,
		}
	}
	return nodes, nil
}

// setupPartitions creates a Raft partition set
func (c *ClusterController) setupPartitions() error {
	if err := c.createPartitionSet(); err != nil {
		return err
	}
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
			Namespace: c.clusterID,
		},
		Spec: v1alpha1.PartitionSetSpec{
			Partitions: c.config.Partitions,
			Template: v1alpha1.PartitionTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"resource": "raft",
					},
				},
				Spec: v1alpha1.PartitionSpec{
					Size:     int32(c.config.PartitionSize),
					Protocol: "raft",
					Image:    "atomix/atomix-raft-protocol:latest",
					Config:   string(bytes),
				},
			},
		},
	}
	_, err = c.atomixclient.K8sV1alpha1().PartitionSets(c.clusterID).Create(set)
	return err
}

// awaitPartitionsReady waits for Raft partitions to complete startup
func (c *ClusterController) awaitPartitionsReady() error {
	for {
		set, err := c.atomixclient.K8sV1alpha1().PartitionSets(c.clusterID).Get("raft", metav1.GetOptions{})
		if err != nil {
			return err
		} else if int(set.Status.ReadyPartitions) == set.Spec.Partitions {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}
