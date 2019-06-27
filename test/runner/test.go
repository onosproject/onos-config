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
	"errors"
	"github.com/onosproject/onos-config/test/env"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	log "k8s.io/klog"
	"strings"
	"time"
)

// TestStatus test status
type TestStatus string

const (
	// TestRunning running
	TestRunning TestStatus = "RUNNING"

	// TestPassed passed
	TestPassed TestStatus = "PASSED"

	// TestFailed failed
	TestFailed TestStatus = "FAILED"
)

// TestRecord contains information about a test run
type TestRecord struct {
	TestID   string
	Args     []string
	Status   TestStatus
	Message  string
	ExitCode int
}

// startTests starts running a test job
func (c *ClusterController) startTests(testID string, tests []string, timeout time.Duration) (corev1.Pod, error) {
	if err := c.createTestJob(testID, tests, timeout); err != nil {
		return corev1.Pod{}, err
	}
	return c.awaitTestJobRunning(testID)
}

// createTestJob creates the job to run tests
func (c *ClusterController) createTestJob(testID string, args []string, timeout time.Duration) error {
	log.Infof("Starting test job %s", testID)

	deviceIds, err := c.getDeviceIds()
	if err != nil {
		return err
	}

	one := int32(1)
	timeoutSeconds := int64(timeout / time.Second)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testID,
			Namespace: c.clusterID,
			Annotations: map[string]string{
				"test-args": strings.Join(args, ","),
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:           &one,
			Completions:           &one,
			BackoffLimit:          &one,
			ActiveDeadlineSeconds: &timeoutSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"cluster":  c.clusterID,
						"test":     testID,
						"resource": testID,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "test",
							Image:           "onosproject/onos-config-integration-tests:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            args,
							Env: []corev1.EnvVar{
								{
									Name:  env.TestDevicesEnv,
									Value: strings.Join(deviceIds, ","),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "secret",
									MountPath: "/etc/onos-config/certs",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
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

	_, err = c.kubeclient.BatchV1().Jobs(c.clusterID).Create(job)
	return err
}

// awaitTestJobRunning blocks until the test job creates a pod in the RUNNING state
func (c *ClusterController) awaitTestJobRunning(testID string) (corev1.Pod, error) {
	log.Infof("Waiting for test job %s to become ready", testID)
	for {
		pod, err := c.getPod(testID)
		if err == nil {
			return pod, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// getStatus gets the status message and exit code of the given pod
func (c *ClusterController) getStatus(pod corev1.Pod) (string, int, error) {
	for {
		obj, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return "", 0, err
		}
		state := obj.Status.ContainerStatuses[0].State
		if state.Terminated != nil {
			return state.Terminated.Message, int(state.Terminated.ExitCode), nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// GetHistory returns the history of test runs on the cluster
func (c *ClusterController) GetHistory() ([]TestRecord, error) {
	jobs, err := c.kubeclient.BatchV1().Jobs(c.clusterID).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	records := make([]TestRecord, 0, len(jobs.Items))
	for _, job := range jobs.Items {
		record, err := c.getRecord(job)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

// GetRecord returns a single record for the given test
func (c *ClusterController) GetRecord(testID string) (TestRecord, error) {
	job, err := c.kubeclient.BatchV1().Jobs(c.clusterID).Get(testID, metav1.GetOptions{})
	if err != nil {
		return TestRecord{}, err
	}
	return c.getRecord(*job)
}

// GetRecord returns a single record for the given test
func (c *ClusterController) getRecord(job batchv1.Job) (TestRecord, error) {
	testID := job.Labels["test"]

	var args []string
	testArgs, ok := job.Annotations["test-args"]
	if ok {
		args = strings.Split(testArgs, ",")
	} else {
		args = make([]string, 0)
	}

	pod, err := c.getPod(testID)
	if err != nil {
		return TestRecord{}, nil
	}

	record := TestRecord{
		TestID: testID,
		Args:   args,
	}

	state := pod.Status.ContainerStatuses[0].State
	if state.Terminated != nil {
		record.Message = state.Terminated.Message
		record.ExitCode = int(state.Terminated.ExitCode)
		if record.ExitCode == 0 {
			record.Status = TestPassed
		} else {
			record.Status = TestFailed
		}
	} else {
		record.Status = TestRunning
	}

	return record, nil
}

// getPod finds the Pod for the given test
func (c *ClusterController) getPod(testID string) (corev1.Pod, error) {
	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: "test=" + testID,
	})
	if err != nil {
		return corev1.Pod{}, err
	} else if len(pods.Items) > 0 {
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodRunning && len(pod.Status.ContainerStatuses) > 0 && pod.Status.ContainerStatuses[0].Ready {
				return pod, nil
			}
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				return pod, nil
			}
		}
	}
	return corev1.Pod{}, errors.New("cannot locate test pod for test " + testID)
}

// getDeviceIds returns a slice of configured simulator device IDs
func (c *ClusterController) getDeviceIds() ([]string, error) {
	devices := []string{}

	// Load the cluster configuration
	config, err := c.config.load()
	if err != nil {
		return nil, err
	}

	// Add devices from the device store
	deviceStoreObj, ok := config["deviceStore"].(map[string]interface{})
	if ok {
		for name := range deviceStoreObj["Store"].(map[string]interface{}) {
			devices = append(devices, name)
		}
	}

	// Get a list of simulators deployed in the cluster
	simulators, err := c.GetSimulators()
	if err != nil {
		return nil, err
	}

	// Add each simulator to the devices
	for _, name := range simulators {
		devices = append(devices, name)
	}
	return devices, nil
}
