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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	atomixk8s "github.com/atomix/atomix-k8s-controller/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	log "k8s.io/klog"
	"net/http"
	"os"
	"time"
)

// ClusterController manages a single cluster in Kubernetes
type ClusterController struct {
	clusterID        string
	restconfig       *rest.Config
	kubeclient       *kubernetes.Clientset
	atomixclient     *atomixk8s.Clientset
	extensionsclient *apiextension.Clientset
	config           *ClusterConfig
}

// Setup sets up a test cluster with the given configuration
func (c *ClusterController) Setup() error {
	log.Infof("Setting up test cluster %s", c.clusterID)
	if err := c.setupAtomixController(); err != nil {
		return err
	}
	if err := c.setupPartitions(); err != nil {
		return err
	}
	if err := c.setupOnosConfig(); err != nil {
		return err
	}
	return nil
}

// AddSimulator adds a device simulator with the given configuration
func (c *ClusterController) AddSimulator(name string, config *SimulatorConfig) error {
	log.Infof("Setting up simulator %s/%s", name, c.clusterID)
	if err := c.setupSimulator(name, config); err != nil {
		return err
	}

	log.Infof("Waiting for simulator %s/%s to become ready", name, c.clusterID)
	if err := c.awaitSimulatorReady(name); err != nil {
		return err
	}

	log.Infof("Adding simulator %s/%s to onos-config nodes", name, c.clusterID)
	if err := c.addSimulatorToConfig(name); err != nil {
		return err
	}
	return nil
}

// RunTests runs the given tests on Kubernetes
func (c *ClusterController) RunTests(testID string, tests []string, timeout time.Duration) (string, int, error) {
	// Default the test timeout to 10 minutes
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	// Start the test job
	pod, err := c.startTests(testID, tests, timeout)
	if err != nil {
		return "", 0, err
	}

	// Stream the logs to stdout
	if err = c.streamLogs(pod); err != nil {
		return "", 0, err
	}

	// Get the exit message and code
	return c.getStatus(pod)
}

// GetLogs returns the logs for a test resource
func (c *ClusterController) GetLogs(resourceID string) ([][]string, error) {
	pod, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(resourceID, metav1.GetOptions{})
	if err == nil {
		return c.getAllLogs([]corev1.Pod{*pod})
	} else if !k8serrors.IsNotFound(err) {
		return nil, err
	}

	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: "resource=" + resourceID,
	})
	if err != nil {
		return nil, err
	} else if len(pods.Items) == 0 {
		return nil, errors.New("unknown test resource " + resourceID)
	} else {
		return c.getAllLogs(pods.Items)
	}
}

// getAllLogs gets the logs from all of the given pods
func (c *ClusterController) getAllLogs(pods []corev1.Pod) ([][]string, error) {
	allLogs := make([][]string, len(pods))
	for i, pod := range pods {
		logs, err := c.getLogs(pod)
		if err != nil {
			return nil, err
		}
		allLogs[i] = logs
	}
	return allLogs, nil
}

// getLogs gets the logs from the given pod
func (c *ClusterController) getLogs(pod corev1.Pod) ([]string, error) {
	req := c.kubeclient.CoreV1().Pods(c.clusterID).GetLogs(pod.Name, &corev1.PodLogOptions{})
	readCloser, err := req.Stream()
	if err != nil {
		return nil, err
	}

	defer readCloser.Close()

	logs := []string{}
	scanner := bufio.NewScanner(readCloser)
	for scanner.Scan() {
		logs = append(logs, scanner.Text())
	}
	return logs, nil
}

// PortForward forwards a local port to the given remote port on the given resource
func (c *ClusterController) PortForward(resourceId string, localPort int, remotePort int) error {
	pod, err := c.kubeclient.CoreV1().Pods(c.ClusterId).Get(resourceId, metav1.GetOptions{})
	if err != nil {
		return err
	}

	req := c.kubeclient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	roundTripper, upgradeRoundTripper, err := spdy.RoundTripperFor(c.restconfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgradeRoundTripper, &http.Client{Transport: roundTripper}, http.MethodPost, req.URL())

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)
	out, errOut := new(bytes.Buffer), new(bytes.Buffer)

	forwarder, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, remotePort)}, stopChan, readyChan, out, errOut)
	if err != nil {
		return err
	}

	go func() {
		for range readyChan { // Kubernetes will close this channel when it has something to tell us.
		}
		if len(errOut.String()) != 0 {
			fmt.Println(errOut.String())
			os.Exit(1)
		} else if len(out.String()) != 0 {
			fmt.Println(out.String())
		}
	}()

	return forwarder.ForwardPorts()
}

// RemoveSimulator removes a device simulator with the given name
func (c *ClusterController) RemoveSimulator(name string) error {
	log.Infof("Tearing down simulator %s/%s", name, c.clusterID)
	if err := c.teardownSimulator(name); err != nil {
		return err
	}

	log.Infof("Removing simulator %s/%s from onos-config nodes", name, c.clusterID)
	if err := c.removeSimulatorFromConfig(name); err != nil {
		return err
	}
	return nil
}
