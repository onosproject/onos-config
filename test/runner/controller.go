package runner

import (
	"errors"
	"fmt"
	"github.com/atomix/atomix-k8s-controller/pkg/apis/k8s/v1alpha1"
	atomixk8s "github.com/atomix/atomix-k8s-controller/pkg/client/clientset/versioned"
	raft "github.com/atomix/atomix-k8s-controller/proto/atomix/protocols/raft"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	log "k8s.io/klog"
	"os"
	"path/filepath"
	"time"
)

func init() {
	log.SetOutput(os.Stdout)
}

// Controller runs tests on a specific platform
type Controller interface {
	// Runs the given tests
	Run(tests []string)
}

// NewKubeController creates a new Kubernetes integration test controller
func NewKubeController(timeout time.Duration) (Controller, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	return GetKubeController(id.String(), timeout)
}

// GetKubeController returns a Kubernetes integration test controller for the given test ID
func GetKubeController(testId string, timeout time.Duration) (Controller, error) {
	testName := getTestName(testId)

	clientset, err := newKubeClient()
	if err != nil {
		return nil, err
	}

	return &kubeController{
		TestId:     testId,
		TestName:   testName,
		kubeclient: clientset,
		timeout:    timeout,
	}, nil
}

// Kubernetes test controller
type kubeController struct {
	TestId       string
	TestName     string
	kubeclient   *kubernetes.Clientset
	atomixclient *atomixk8s.Clientset
	timeout      time.Duration
}

// Run runs the given tests on Kubernetes
func (c *kubeController) Run(tests []string) {
	// Set up k8s resources
	if err := c.setup(); err != nil {
		exitError(err)
	}

	// Start the test job
	pod, err := c.start(tests)
	if err != nil {
		exitError(err)
	}

	if err = c.streamLogs(pod); err != nil {
		exitError(err)
	}

	message, status, err := c.getStatus(pod)
	c.cleanup()
	if err != nil {
		exitError(err)
	} else {
		fmt.Println(message)
		os.Exit(status)
	}
}

// setup sets up the Kubernetes resources required to run tests
func (c *kubeController) setup() error {
	if err := c.createNamespace(); err != nil {
		return err
	}
	if err := c.createClusterRole(); err != nil {
		return err
	}
	if err := c.createClusterRoleBinding(); err != nil {
		return err
	}
	if err := c.createServiceAccount(); err != nil {
		return err
	}
	if err := c.createController(); err != nil {
		return err
	}
	if err := c.createService(); err != nil {
		return err
	}
	if err := c.awaitControllerRunning(); err != nil {
		return err
	}
	if err := c.createPartitionSet(); err != nil {
		return err
	}
	if err := c.awaitPartitionsRunning(); err != nil {
		return err
	}
	return nil
}

// createNamespace creates a uniquely named namespace with which to run tests
func (c *kubeController) createNamespace() error {
	log.Infof("Creating test namespace %s", c.TestName)
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.TestName,
		},
	}
	_, err := c.kubeclient.CoreV1().Namespaces().Create(namespace)
	return err
}

// createClusterRole creates the ClusterRole required by the Atomix controller if not yet created
func (c *kubeController) createClusterRole() error {
	log.Infof("Creating test controller role atomix-controller/%s", c.TestName)
	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
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
					"poddisruptionbudget",
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

// createClusterRoleBinding creates the ClusterRoleBinding required by the Atomix controller for the test namespace
func (c *kubeController) createClusterRoleBinding() error {
	log.Infof("Creating test controller role binding atomix-controller/%s", c.TestName)
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "atomix-controller",
				Namespace: c.TestName,
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
			c.deleteClusterRoleBinding()
			return c.createClusterRoleBinding()
		} else {
			return err
		}
	}
	return nil
}

// createServiceAccount creates a ServiceAccount used by the Atomix controller
func (c *kubeController) createServiceAccount() error {
	log.Infof("Creating test controller service account atomix-controller/%s", c.TestName)
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
		},
	}
	_, err := c.kubeclient.CoreV1().ServiceAccounts(c.TestName).Create(serviceAccount)
	return err
}

// createController creates the Atomix controller Deployment
func (c *kubeController) createController() error {
	log.Infof("Creating test controller atomix-controller/%s", c.TestName)
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
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
	_, err := c.kubeclient.AppsV1().Deployments(c.TestName).Create(deployment)
	return err
}

// createService creates a service for the controller
func (c *kubeController) createService() error {
	log.Infof("Creating test controller service atomix-controller/%s", c.TestName)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "atomix-controller",
			Namespace: c.TestName,
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
	_, err := c.kubeclient.CoreV1().Services(c.TestName).Create(service)
	return err
}

// awaitControllerRunning blocks until the Atomix controller is in the RUNNING state
func (c *kubeController) awaitControllerRunning() error {
	log.Infof("Awaiting test controller atomix-controller/%s", c.TestName)
	for {
		pods, err := c.kubeclient.CoreV1().Pods(c.TestName).List(metav1.ListOptions{
			LabelSelector: "name=atomix-controller",
		})
		if err != nil {
			return err
		} else if len(pods.Items) > 0 && pods.Items[0].Status.Phase == corev1.PodRunning {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// createPartitionSet creates a Raft partition set
func (c *kubeController) createPartitionSet() error {
	log.Infof("Creating test partition set raft/%s", c.TestName)
	bytes, err := yaml.Marshal(&raft.RaftProtocol{})
	if err != nil {
		return err
	}

	set := &v1alpha1.PartitionSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "raft",
			Namespace: c.TestName,
		},
		Spec: v1alpha1.PartitionSetSpec{
			Partitions: 3,
			Template: v1alpha1.PartitionTemplateSpec{
				Spec: v1alpha1.PartitionSpec{
					Size: 3,
					Protocol: "raft",
					Image: "atomix/atomix-raft-protocol:latest",
					Config: string(bytes),
				},
			},
		},
	}
	_, err = c.atomixclient.K8sV1alpha1().PartitionSets(c.TestName).Create(set)
	return err
}

// awaitPartitionsRunning waits for Raft partitions to complete startup
func (c *kubeController) awaitPartitionsRunning() error {
	log.Infof("Awaiting test partition set raft/%s", c.TestName)
	for {
		set, err := c.atomixclient.K8sV1alpha1().PartitionSets(c.TestName).Get("raft", metav1.GetOptions{})
		if err != nil {
			return err
		} else if int(set.Status.ReadyPartitions) == set.Spec.Partitions {
			return nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// start starts running the test job
func (c *kubeController) start(args []string) (corev1.Pod, error) {
	if err := c.createJob(args); err != nil {
		return corev1.Pod{}, err
	}
	return c.awaitJobRunning()
}

// createJob creates the job to run tests
func (c *kubeController) createJob(args []string) error {
	log.Infof("Starting test job %s", c.TestName)
	one := int32(1)
	timeout := int64(c.timeout / time.Second)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.TestName,
			Namespace: c.TestName,
		},
		Spec: batchv1.JobSpec{
			Parallelism:           &one,
			Completions:           &one,
			BackoffLimit:          &one,
			ActiveDeadlineSeconds: &timeout,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"test": c.TestName,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "atomix-test",
							Image:           "atomix/atomix-integration-tests:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args:            args,
							Env: []corev1.EnvVar{
								{
									Name:  "ATOMIX_CONTROLLER",
									Value: fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", c.TestName),
								},
								{
									Name:  "ATOMIX_APP",
									Value: "test",
								},
								{
									Name:  "ATOMIX_NAMESPACE",
									Value: c.TestName,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.kubeclient.BatchV1().Jobs(c.TestName).Create(job)
	return err
}

// awaitJobRunning blocks until the test job creates a pod in the RUNNING state
func (c *kubeController) awaitJobRunning() (corev1.Pod, error) {
	log.Infof("Awaiting test job %s", c.TestName)
	for {
		pods, err := c.kubeclient.CoreV1().Pods(c.TestName).List(metav1.ListOptions{
			LabelSelector: "test=" + c.TestName,
		})
		if err != nil {
			return corev1.Pod{}, err
		} else if len(pods.Items) > 0 {
			for _, pod := range pods.Items {
				if pod.Status.Phase == corev1.PodRunning {
					return pod, nil
				}
			}
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// streamLogs streams the logs from the given pod to stdout
func (c *kubeController) streamLogs(pod corev1.Pod) error {
	req := c.kubeclient.CoreV1().Pods(c.TestName).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow: true,
	})
	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	defer readCloser.Close()

	buf := make([]byte, 1024)
	for {
		n, err := readCloser.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Print(string(buf[:n]))
	}
	return nil
}

// getStatus gets the status message and exit code of the given pod
func (c *kubeController) getStatus(pod corev1.Pod) (string, int, error) {
	for {
		obj, err := c.kubeclient.CoreV1().Pods(c.TestName).Get(pod.Name, metav1.GetOptions{})
		if err != nil {
			return "", 0, err
		} else {
			state := obj.Status.ContainerStatuses[0].State
			if state.Terminated != nil {
				return state.Terminated.Message, int(state.Terminated.ExitCode), nil
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// cleanup deletes test resources from the Kubernetes cluster
func (c *kubeController) cleanup() error {
	if err := c.deleteNamespace(); err != nil {
		return err
	}
	if err := c.deleteClusterRoleBinding(); err != nil {
		return err
	}
	return nil
}

// deleteClusterRoleBinding deletes the ClusterRoleBinding used by the test
func (c *kubeController) deleteClusterRoleBinding() error {
	return c.kubeclient.RbacV1().ClusterRoleBindings().Delete("atomix-controller", &metav1.DeleteOptions{})
}

// deleteNamespace deletes the Namespace used by the test and all resources within it
func (c *kubeController) deleteNamespace() error {
	return c.kubeclient.CoreV1().Namespaces().Delete(c.TestName, &metav1.DeleteOptions{})
}

// getTestName returns a qualified test name derived from the given test ID suitable for use in k8s resource names
func getTestName(testId string) string {
	return "atomix-test-" + testId
}

// exitError prints the given err to stdout and exits with exit code 1
func exitError(err error) {
	fmt.Println(err)
	os.Exit(1)
}

// newKubeClient returns a new Kubernetes client from the environment
func newKubeClient() (*kubernetes.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		if home == "" {
			return nil, errors.New("no home directory configured")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	return kubernetes.NewForConfig(config)
}

// newAtomixKubeClient returns a new Kubernetes client from the environment
func newAtomixKubeClient() (*atomixk8s.Clientset, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := homeDir()
		if home == "" {
			return nil, errors.New("no home directory configured")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	return atomixk8s.NewForConfig(config)
}

// homeDir returns the user's home directory if defined by environment variables
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
