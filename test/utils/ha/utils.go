// Copyright 2020-present Open Networking Foundation.
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

package ha

import (
	"strings"
	"testing"

	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	v1 "github.com/onosproject/helmit/pkg/kubernetes/core/v1"
	"github.com/stretchr/testify/assert"
)

const (
	onosComponentName = "onos-umbrella"
)

// GetPodListOrFail gets the list of pods active in the onos-config release. The test is failed if getting the list returns
// an error.
func GetPodListOrFail(t *testing.T) []*v1.Pod {
	release := helm.Chart(onosComponentName).Release(onosComponentName)
	client := kubernetes.NewForReleaseOrDie(release)
	podList, err := client.
		CoreV1().
		Pods().
		List()
	assert.NoError(t, err)
	return podList
}

// CrashPodOrFail deletes the given pod and fails the test if there is an error
func CrashPodOrFail(t *testing.T, pod *v1.Pod) {
	err := pod.Delete()
	assert.NoError(t, err)
}

// FindPodWithPrefix looks for the first pod whose name matches the given prefix string. The test is failed
// if no matching pod is found.
func FindPodWithPrefix(t *testing.T, prefix string) *v1.Pod {
	podList := GetPodListOrFail(t)
	for _, p := range podList {
		if strings.HasPrefix(p.Name, prefix) {
			return p
		}
	}
	assert.Failf(t, "No pod found matching %s", prefix)
	return nil
}
