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
	"github.com/atomix/atomix-k8s-controller/pkg/apis/k8s/v1alpha1"
	raft "github.com/atomix/atomix-k8s-controller/proto/atomix/protocols/raft"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "k8s.io/klog"
	"time"
)

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
