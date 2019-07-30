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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	atomixk8s "github.com/atomix/atomix-k8s-controller/pkg/client/clientset/versioned"
	"github.com/onosproject/onos-config/test/console"
	corev1 "k8s.io/api/core/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// ClusterController manages a single cluster in Kubernetes
type ClusterController struct {
	clusterID        string
	restconfig       *rest.Config
	kubeclient       *kubernetes.Clientset
	atomixclient     *atomixk8s.Clientset
	extensionsclient *apiextension.Clientset
	config           *ClusterConfig
	status           *console.StatusWriter
}

// Setup sets up a test cluster with the given configuration
func (c *ClusterController) Setup() console.ErrorStatus {
	c.status.Start("Setting up Atomix controller")
	if err := c.setupAtomixController(); err != nil {
		return c.status.Fail(err)
	}
	c.status.Succeed()
	c.status.Start("Starting Raft partitions")
	if err := c.setupPartitions(); err != nil {
		return c.status.Fail(err)
	}
	c.status.Succeed()
	c.status.Start("Creating secret for onos nodes")
	if err := c.createOnosSecret(); err != nil {
		return c.status.Fail(err)
	}
	c.status.Succeed()
	c.status.Start("Bootstrapping onos-config cluster")
	if err := c.setupOnosConfig(); err != nil {
		return c.status.Fail(err)
	}
	c.status.Start("Bootstrapping onos-topo cluster")
	if err := c.setupOnosTopo(); err != nil {
		return c.status.Fail(err)
	}
	c.status.Succeed()
	return c.status.Succeed()
}

// AddSimulator adds a device simulator with the given configuration
func (c *ClusterController) AddSimulator(name string, config *SimulatorConfig) console.ErrorStatus {
	c.status.Start("Setting up simulator")
	if err := c.setupSimulator(name, config); err != nil {
		return c.status.Fail(err)
	}
	c.status.Start("Reconfiguring onos-config nodes")
	if err := c.addSimulatorToConfig(name); err != nil {
		return c.status.Fail(err)
	}
	return c.status.Succeed()
}

// AddNetwork adds a stratum network with the given configuration
func (c *ClusterController) AddNetwork(name string, config *NetworkConfig) console.ErrorStatus {
	c.status.Start("Setting up network")
	if err := c.setupNetwork(name, config); err != nil {
		return c.status.Fail(err)
	}
	c.status.Start("Reconfiguring onos-config nodes")
	if err := c.addNetworkToConfig(name, config); err != nil {
		return c.status.Fail(err)
	}
	return c.status.Succeed()
}

// RunTests runs the given tests on Kubernetes
func (c *ClusterController) RunTests(testID string, tests []string, timeout time.Duration) (string, int, console.ErrorStatus) {
	// Default the test timeout to 10 minutes
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	// Start the test job
	c.status.Start("Starting test job: " + testID)
	pod, err := c.startTests(testID, tests, timeout)
	if err != nil {
		return "", 0, c.status.Fail(err)
	}
	c.status.Succeed()

	// Get the stream of logs for the pod
	reader, err := c.streamLogs(pod)
	if err != nil {
		return "", 0, c.status
	}
	defer reader.Close()

	// Stream the logs to stdout
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", 0, c.status
		}
		fmt.Print(string(buf[:n]))
	}

	// Get the exit message and code
	message, status, err := c.getStatus(pod)
	if err != nil {
		return "failed to retrieve exit code", 1, c.status
	}
	return message, status, c.status
}

// GetResources returns a list of resource IDs matching the given resource name
func (c *ClusterController) GetResources(name string) ([]string, error) {
	pod, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(name, metav1.GetOptions{})
	if err == nil {
		return []string{pod.Name}, nil
	} else if !k8serrors.IsNotFound(err) {
		return nil, err
	}

	pods, err := c.kubeclient.CoreV1().Pods(c.clusterID).List(metav1.ListOptions{
		LabelSelector: "resource=" + name,
	})
	if err != nil {
		return nil, err
	} else if len(pods.Items) == 0 {
		return nil, errors.New("unknown test resource " + name)
	}

	resources := make([]string, len(pods.Items))
	for i, pod := range pods.Items {
		resources[i] = pod.Name
	}
	return resources, nil
}

// GetLogs returns the logs for a single test resource
func (c *ClusterController) GetLogs(resourceID string, options corev1.PodLogOptions) ([]byte, error) {
	pod, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(resourceID, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return c.getLogs(*pod, options)
}

// getLogs gets the logs from the given pod
func (c *ClusterController) getLogs(pod corev1.Pod, options corev1.PodLogOptions) ([]byte, error) {
	req := c.kubeclient.CoreV1().Pods(c.clusterID).GetLogs(pod.Name, &options)
	readCloser, err := req.Stream()
	if err != nil {
		return nil, err
	}

	defer readCloser.Close()

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(readCloser); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// StreamLogs streams the logs for the given test resources to stdout
func (c *ClusterController) StreamLogs(resourceID string) (io.ReadCloser, error) {
	pod, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(resourceID, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return c.streamLogs(*pod)
}

// streamLogs streams the logs from the given pod to stdout
func (c *ClusterController) streamLogs(pod corev1.Pod) (io.ReadCloser, error) {
	req := c.kubeclient.CoreV1().Pods(c.clusterID).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow: true,
	})
	return req.Stream()
}

// DownloadLogs downloads the logs for the given resource to the given path
func (c *ClusterController) DownloadLogs(resourceID string, path string, options corev1.PodLogOptions) console.ErrorStatus {
	c.status.Start("Downloading logs")
	pod, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(resourceID, metav1.GetOptions{})
	if err != nil {
		return c.status.Fail(err)
	}
	if err := c.downloadLogs(*pod, path, options); err != nil {
		return c.status.Fail(err)
	}
	return c.status.Succeed()
}

// downloadLogs downloads the logs from the given pod to the given path
func (c *ClusterController) downloadLogs(pod corev1.Pod, path string, options corev1.PodLogOptions) error {
	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	// Get a stream of logs
	req := c.kubeclient.CoreV1().Pods(c.clusterID).GetLogs(pod.Name, &options)
	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	defer readCloser.Close()

	_, err = io.Copy(file, readCloser)
	return err
}

// PortForward forwards a local port to the given remote port on the given resource
func (c *ClusterController) PortForward(resourceID string, localPort int, remotePort int) error {
	pod, err := c.kubeclient.CoreV1().Pods(c.clusterID).Get(resourceID, metav1.GetOptions{})
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
func (c *ClusterController) RemoveSimulator(name string) console.ErrorStatus {
	c.status.Start("Tearing down simulator")
	if err := c.teardownSimulator(name); err != nil {
		c.status.Fail(err)
	}
	c.status.Start("Reconfiguring onos-config nodes")
	if err := c.removeSimulatorFromConfig(name); err != nil {
		return c.status.Fail(err)
	}
	return c.status.Succeed()
}

// RemoveNetwork removes a stratum network with the given name
func (c *ClusterController) RemoveNetwork(name string) console.ErrorStatus {
	c.status.Start("Tearing down network")
	label := "network=" + name
	configMaps, _ := c.kubeclient.CoreV1().ConfigMaps(c.clusterID).List(metav1.ListOptions{
		LabelSelector: label,
	})

	if err := c.teardownNetwork(name); err != nil {
		c.status.Fail(err)
	}
	c.status.Start("Reconfiguring onos-config nodes")
	if err := c.removeNetworkFromConfig(name, configMaps); err != nil {
		return c.status.Fail(err)
	}
	return c.status.Succeed()
}
