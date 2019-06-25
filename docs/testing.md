# Testing onos-config

The onos-config project provides an 
[integration testing framework](https://github.com/onosproject/onos-config/pull/374) for
running end-to-end tests on [Kubernetes]. This document describes the process for managing 
a development/test environment and running integration tests with `onit`.

## Setup

The integration test framework is designed to operate on a Kubernetes cluster. It's recommended
that users use a local Kubernetes cluster suitable for development, e.g. [Minikube], [kind],
or [MicroK8s].

### Configuration

The test framework is controlled through the `onit` command. To install the `onit` command,
use `go get`:

```bash
> go get github.com/onosproject/onos-config/test/cmd/onit
```

To interact with a Kubernetes cluster, the `onit` command must have access to a local
Kubernetes configuration. Onit expects the same configuration as `kubectl` and will connect
to the same Kubernetes cluster as `kubectl` will connect to, so to determine which Kubernetes 
cluster onit will use, simply run `kubectl cluster-info`:

```bash
> kubectl cluster-info
Kubernetes master is running at https://127.0.0.1:49760
KubeDNS is running at https://127.0.0.1:49760/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
```

See the [Kubernetes documentation](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
for details on configuring both `kubectl` and `onit` to connect to a new cluster or multiple
clusters.

The `onit` command also maintains some cluster metadata in a local configuration file. The search
path for the `onit.yaml` configuration file is:
* `~/.onos`
* `/etc/onos`
* `.`

Users do not typically need to modify the `onit.yaml` configuration file directly. The onit 
configuration is primarily managed through various onit commands like `onit set`, `onit create`,
`onit add`, etc. It's recommended that users avoid modifying the onit configuration
file, but it's nevertheless important to note that the application must have write access to one
of the above paths.

### Docker

The `onit` command manages clusters and runs tests by deploying locally built [Docker] containers
on [Kubernetes]. Docker image builds are an essential component of the `onit` workflow. **Each time a
change is made to either the core or integration tests, Docker images must be rebuilt** and made
available within the Kubernetes cluster in which tests are being run. The precise process for building
Docker images and adding them to a local Kubernetes cluster is different for each setup.

#### Building for Minikube

[Minikube] runs a VM with its own Docker daemon running inside it. To build the Docker images
for Minikube, ensure you use configure your shell to use Minikube Docker context before building:

```bash
> eval $(minikube docker-env)
```

Once the shell has been configured, use `make images` to build the Docker images:

```bash
> make images
```

Note that `make images` _must be run every time a change is made_ to either the core code
or integration tests.

#### Building for Kind

[Kind][kind] provides an alternative to [Minikube] which runs Kubernetes in a Docker container.
As with Minikube, kind requires specific setup to ensure Docker images modified and built
locally can be run within the kind cluster. Rather than switching your Docker environment to
a remote Docker server, kind requires that images be explicitly loaded into the cluster each
time they're built. For this reason, we provide a convenience make target: `kind`:

```bash
> make kind
```

When the `kind` target is run, the `onos-config` and `onos-config-integration-tests` images will
be built and loaded into the kind cluster, so no additional step is necessary.

## Usage

The primary interface for setting up test clusters and running tests is the `onit` command,
which provides a suite of commands for setting up and tearing down test clusters, adding
and removing [device simulators][simulators], running tests, and viewing test history.

### Cluster Setup

The first step to running tests is to setup a test cluster with `onit create cluster`:

```bash
> onit create cluster
I0624 15:36:33.569276   65347 controller.go:100] Setting up test cluster 84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:33.569309   65347 controller.go:175] Setting up test namespace onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:33.592048   65347 controller.go:187] Setting up Atomix controller atomix-controller/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:33.720107   65347 controller.go:210] Waiting for Atomix controller atomix-controller/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941 to become ready
I0624 15:36:47.272478   65347 controller.go:527] Setting up partitions raft/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:47.287600   65347 controller.go:532] Waiting for partitions raft/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941 to become ready
84b9795a-96d0-11e9-bd2a-784f43889941
```

To setup the cluster, onit creates a unique namespace within which to create test resources,
deploys [Atomix] inside the test namespace, and configures and deploys onos-config nodes.
Once the cluster is setup, the command will output the name of the test namespace. The namespace
can be used to view test resources via `kubectl`:

```bash
> kubectl get pods -n 84b9795a-96d0-11e9-bd2a-784f43889941
atomix-controller-77b78fcc54-qzzps                     1/1     Running     0          11h
onos-config-786594d4b5-tqwwr                           1/1     Running     0          29m
raft-1-0                                               1/1     Running     0          11h
```

The `create cluster` command supports additional flags for defining the cluster architecture:
* `--config`/`-c` - the configuration with which to bootstrap `onos-config` node stores. The configuration
must reference a JSON file name in `test/configs/store`, e.g. `default` refers to `default.json`
* `--nodes`/`-n` - the number of `onos-config` nodes to deploy
* `--partitions`/`-p` - the number of Raft partitions to create for stores
* `--partition-size`/`-s` - the size of each Raft partition

Once the cluster is setup, the cluster configuration will be added to the `onit` configuration
and the deployed cluster will be set as the current cluster context:

```bash
> onit get cluster
84b9795a-96d0-11e9-bd2a-784f43889941
> onit get clusters
ID                                     SIZE   PARTITIONS
84b9795a-96d0-11e9-bd2a-784f43889941   1      1
0e2ad27a-9720-11e9-ad72-acde48001122   1      1
```

When multiple clusters are deployed, you can switch between clusters by setting the current
cluster context:

```bash
> onit set cluster 0e2ad27a-9720-11e9-ad72-acde48001122
0e2ad27a-9720-11e9-ad72-acde48001122
```

This will run all future cluster operations on the configured cluster. Alternatively, most
commands support a flag to override the default cluster:

```bash
> onit get history -c 0e2ad27a-9720-11e9-ad72-acde48001122
...
```

To delete a cluster, run `onit delete cluster`:
```bash
> onit delete cluster
```

### Adding Devices

Most tests require devices to be added to the cluster. The `onit` command supports adding and
removing [device simulators][simulators] through the `add` and `remove` commands. To add a
simulator to the current cluster, simply run `onit add simulator`:

```bash
> onit add simulator 
I0625 12:01:54.498233   43235 controller.go:110] Setting up simulator device-2996584472/onos-cluster-0e2ad27a-9720-11e9-ad72-acde48001122
I0625 12:01:54.602006   43235 controller.go:115] Waiting for simulator device-2996584472/onos-cluster-0e2ad27a-9720-11e9-ad72-acde48001122 to become ready
I0625 12:02:03.107258   43235 controller.go:1034] Redeploying onos-config cluster onos-config/onos-cluster-0e2ad27a-9720-11e9-ad72-acde48001122
I0625 12:02:03.527489   43235 controller.go:1047] Waiting for onos-config cluster onos-config/onos-cluster-0e2ad27a-9720-11e9-ad72-acde48001122 to become ready
device-2996584472
```

When a simulator is added to the cluster, the cluster is reconfigured in two phases:
* Bootstrap a new [device simulator][simulators] with the provided configuration
* Reconfigure and redeploy the onos-config cluster with the new device in its stores

Simulators can similarly be removed with the `remove simulator` command:

```bash
> onit remove simulator device-2996584472
```

As with the `add` command, removing a simulator requires that the onos-config cluster be reconfigured
and redeployed.

### Running Tests

Once the cluster has been setup for the test, to run a test simply use `onit run`:

```bash
> onit run test-single-path-test
I0625 12:07:14.958790   43603 controller.go:1080] Starting test job onos-test-71a0623c-977c-11e9-8478-acde48001122
I0625 12:07:14.982862   43603 controller.go:1147] Waiting for test job 71a0623c-977c-11e9-8478-acde48001122 to become ready
=== RUN   test-single-path-test
--- PASS: test-single-path-test (0.04s)
PASS
```

You can specify as many tests as desired:

```bash
> onit run test-single-path-test test-multi-path-test
...
```

Each test run is recorded as a job in the Kubernetes cluster. This ensures that logs, statuses,
and exit codes are retained for the lifetime of the cluster. Onit supports viewing past test
runs and logs via the `get` command:

```bash
> onit get history
ID                                     TESTS                   STATUS   EXIT CODE   MESSAGE
3cf7311a-9776-11e9-bfc3-acde48001122   test-integration-test   PASSED   0
68ad9154-977c-11e9-bcf2-acde48001122   test-integration-test   FAILED   1
71a0623c-977c-11e9-8478-acde48001122   test-single-path-test   PASSED   0
9e512cdc-9720-11e9-ba6e-acde48001122   *                       PASSED   0
da629d06-9774-11e9-bb50-acde48001122   *                       PASSED   0
```

To get the logs from a specific test, use `onit get logs` with the test ID:

```bash
> onit get logs 71a0623c-977c-11e9-8478-acde48001122
=== RUN   test-single-path-test
--- PASS: test-single-path-test (0.04s)
PASS
```

### Debugging

The `onit` command provides a set of commands for debugging test clusters. The `onit` command
can be used to `get logs` for every resource deployed in the test cluster. Simply pass the
resource ID (e.g. test `ID`, node `ID`, partition `ID`, etc) to the `onit get logs` command
to get the logs for a resource.

To list the onos-config nodes running in the cluster, use `onit get nodes`:

```bash
> onit get nodes
ID                            STATUS
onos-config-f5b7758dc-rdbqt   RUNNING
> onit get logs onos-config-f5b7758dc-rdbqt
I0625 21:55:32.027255       1 onos-config.go:114] Starting onos-config
I0625 21:55:32.030184       1 manager.go:98] Configuration store loaded from /etc/onos-config/configs/configStore.json
I0625 21:55:32.030358       1 manager.go:105] Change store loaded from /etc/onos-config/configs/changeStore.json
I0625 21:55:32.031087       1 manager.go:112] Device store loaded from /etc/onos-config/configs/deviceStore.json
I0625 21:55:32.031222       1 manager.go:119] Network store loaded from /etc/onos-config/configs/networkStore.json
I0625 21:55:32.031301       1 manager.go:47] Creating Manager
...
```

To list the Raft partitions running in the cluster, use `onit get partitions`:

```bash
> onit get partitions
ID   GROUP   NODES
1    raft    raft-1-0
> onit get logs raft-1-0
21:10:24.466 [main] INFO  io.atomix.server.AtomixServerRunner - Node ID: raft-1-0
21:10:24.472 [main] INFO  io.atomix.server.AtomixServerRunner - Partition Config: /etc/atomix/partition.json
21:10:24.472 [main] INFO  io.atomix.server.AtomixServerRunner - Protocol Config: /etc/atomix/protocol.json
21:10:24.473 [main] INFO  io.atomix.server.AtomixServerRunner - Starting server
...
```

To list the tests that have been run, use `onit get history`:

```bash
> onit get history
ID                                     TESTS                   STATUS   EXIT CODE   MESSAGE
3cf7311a-9776-11e9-bfc3-acde48001122   test-integration-test   PASSED   0
68ad9154-977c-11e9-bcf2-acde48001122   test-integration-test   FAILED   1
71a0623c-977c-11e9-8478-acde48001122   test-single-path-test   PASSED   0
9e512cdc-9720-11e9-ba6e-acde48001122   *                       PASSED   0
da629d06-9774-11e9-bb50-acde48001122   *                       PASSED   0
> onit get logs 71a0623c-977c-11e9-8478-acde48001122
=== RUN   test-single-path-test
--- PASS: test-single-path-test (0.04s)
PASS
```

## API

Tests are implemented using Go's `testing` package;

```go
func MyTest(t *testing.T) {
	t.Fail("you messed up!")
}
```

However, rather than running tests using `go test`, we provide a custom registry of tests to
allow human-readable names to be assigned to tests for ease of use. Once you've written a test,
register the test in an `init` function:

```go
func init() {
	Registry.Register("my-test", MyTest)
}
```

Once a test has been registered, you should be able to see the test via the `onit` command:

```bash
> onit list
my-test
...
```

The test framework provides utility functions for creating clients and other resources within
the test environment. The test environment is provided by the `env` package:

```go
client, err := env.NewGnmiClient(context.Background(), "")
...
```

When devices are deployed in the test configuration, a list of device IDs can be retrieved from
the environment:

```go
devices := env.GetDevices()
```

[Kubernetes]: https://kubernetes.io
[Minikube]: https://kubernetes.io/docs/setup/learning-environment/minikube/
[kind]: https://github.com/kubernetes-sigs/kind
[MicroK8s]: https://microk8s.io/
[Docker]: https://www.docker.com/
[Atomix]: https://atomix.io
[simulators]: https://github.com/onosproject/simulators
