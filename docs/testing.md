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


### Onit Auto-Completion
*Onit* supports shell auto-completion for its various commands, sub-commands and flags.
You can enable this feature for *bash* or *zsh* as follows:
#### Bash Auto-Completion
To enable this for **bash**, run the following from the shell:

```bash
> eval "$(onit completion bash)"
```
#### Zsh Auto-Completion 

To enable this for **zsh**, run the following from the shell:
```bash
> source <(onit completion zsh)
```

**Note**: We also recomment to add the output of the above commands to *.bashrc* or *.zshrc*.

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

Assuming you have dowloaded kind as per [instructions][kind-install], the first time you boot the kind cluster 
or if you have rebooted your docker deamon you need to issue:

```bash
> kind create cluster
```

and for each window you intend to use `onit` commands in you will need to export the `KUBECONFIG` 
variable:

```bash
> export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
```

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
and removing [device simulators][simulators], adding and removing networks of [stratum] switches, running tests, and viewing test history. To see list of `onit commands` run `onit` from the shell as follows:
```bash
> onit 
Usage:
  onit [command]

Available Commands:
  add         Add resources to the cluster
  completion  Generated bash or zsh auto-completion script
  create      Create a test resource on Kubernetes
  debug       Open a debugger port to the given resource
  delete      Delete Kubernetes test resources
  fetch       Fetch resources from the cluster
  get         Get test configurations
  help        Help about any command
  remove      Remove resources from the cluster
  run         Run integration tests
  set         Set test configurations

Flags:
  -h, --help   help for onit

Use "onit [command] --help" for more information about a command.
```

### Cluster Setup

The first step to running tests is to setup a test cluster with `onit create cluster`:

```bash
> onit create cluster
 ✓ Creating cluster namespace
 ✓ Setting up Atomix controller
 ✓ Starting Raft partitions
 ✓ Bootstrapping onos-config cluster
cluster-b8c45834-a81c-11e9-82f4-3c15c2cff232
```

To setup the cluster, onit creates a unique namespace within which to create test resources,
deploys [Atomix] inside the test namespace, and configures and deploys onos-config nodes.
Once the cluster is setup, the command will output the name of the test namespace. The namespace
can be used to view test resources via `kubectl`:

```bash
> kubectl get pods -n cluster-b8c45834-a81c-11e9-82f4-3c15c2cff232
NAME                                 READY   STATUS    RESTARTS   AGE
atomix-controller-555dd58f9f-nsx5g   1/1     Running   0          87s
onos-config-66d54956f5-xwpsh         1/1     Running   0          52s
raft-1-0                             1/1     Running   0          73s
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
cluster-b8c45834-a81c-11e9-82f4-3c15c2cff232
```

You can also create a cluster by passing a name to the `onit create cluster` command. To create a cluster with name `onit-2`, we run the following command:

```bash
> onit create cluster onit-2
 ✓ Creating cluster namespace
 ✓ Setting up Atomix controller
 ✓ Starting Raft partitions
 ✓ Bootstrapping onos-config cluster
onit-2
```

if we run `onit get clusters` command, we should be able to see the two clusters that we created:

```bash
> onit get clusters
ID                                             SIZE   PARTITIONS
cluster-b8c45834-a81c-11e9-82f4-3c15c2cff232   1      1
onit-2
```

When multiple clusters are deployed, you can switch between clusters by setting the current
cluster context:

```bash
> onit set cluster onit-2
onit-2
```

This will run all future cluster operations on the configured cluster. Alternatively, most commands support a flag to override the default cluster:

To delete a cluster, run `onit delete cluster`:
```bash
> onit delete cluster
✓ Deleting cluster namespace
```

### Adding Simulators

Most tests require devices to be added to the cluster. The `onit` command supports adding and
removing [device simulators][simulators] through the `add` and `remove` commands. To add a
simulator to the current cluster, simply run `onit add simulator`:

```bash
> onit add simulator 
✓ Setting up simulator
✓ Reconfiguring onos-config nodes
device-1186885096
```

When a simulator is added to the cluster, the cluster is reconfigured in two phases:
* Bootstrap a new [device simulator][simulators] with the provided configuration
* Reconfigure and redeploy the onos-config cluster with the new device in its stores

To give a name to a simulator, pass a name to `onit add simulator` command as follows
```bash
> onit add simulator sim-2
✓ Setting up simulator
✓ Reconfiguring onos-config nodes
sim-2
```

To get list of simulators, run `onit get simulators` as follows:

```bash
> onit get simulators 
device-1186885096
sim-2
```
Simulators can similarly be removed with the `remove simulator` command:

```bash
> onit remove simulator device-1186885096
 ✓ Tearing down simulator
 ✓ Reconfiguring onos-config nodes
```

As with the `add` command, removing a simulator requires that the onos-config cluster be reconfigured and redeployed.

### Adding Networks
To run some of the tests on stratum switches, we can create a network of stratum switches using Mininet. To create a network of stratum switches, we can use `onit add network [Name] [Mininet Options]` as follows: 

* To create a single node network, simply run `onit add network`. This command creates a single node network and assigns a name to it automatically. 
* To create a linear network topology with two switches and name it *stratum-linear*, simply run the following command:

```bash
> onit add network stratum-linear -- --topo linear,2
✓ Setting up network
✓ Reconfiguring onos-config nodes
stratum-linear
```

When a network is added to the cluster, the cluster is reconfigured in two phases:
* Bootstrap one or more than one new stratum switches with the provided configuration
* Reconfigure and redeploy the onos-config cluster with the new switches in its stores

To add a single node network topology, run the following command:
```bash
> onit add network
✓ Setting up network
✓ Reconfiguring onos-config nodes
network-2878434070
```

To get list of networks, run the following command:
```bash
> onit get networks
network-2878434070
stratum-linear
```
Networks can be removed using `onit remove network` command. For example, to remove the linear topolog that is created using the above command, you should run the following command:
```bash
> onit remove network stratum-linear
 ✓ Tearing down network
 ✓ Reconfiguring onos-config nodes
```

As with the `add` command, removing a network requires that the onos-config cluster be reconfigured and redeployed.

**Note**: In the current implementation, we support the following network topologies:

* A *Single* node network topology
* A *Linear* network topology

### Running Tests

#### Running single Tests

Once the cluster has been setup for the test, to run a test simply use `onit run`:

```bash
> onit run test single-path
 ✓ Starting test job: test-25324770
=== RUN   single-path
--- PASS: single-path (0.46s)
PASS
PASS
```

You can specify as many tests as desired:

```bash
> onit run test single-path transaction subscribe
...
```

#### Running a suite of Tests

`onit` can also run a suite of tests e.g. `integration-tests` which encompasses all the active integration tests.
```bash
> onit run suite integration-tests
 ✓ Starting test job: test-3109317976
=== RUN   single-path
--- PASS: single-path (0.20s)
=== RUN   subscribe
--- PASS: subscribe (0.09s)
PASS
```

#### Test Run logs

Each test run is recorded as a job in the Kubernetes cluster. This ensures that logs, statuses,
and exit codes are retained for the lifetime of the cluster. Onit supports viewing past test
runs and logs via the `get` command:

```bash
> onit get history
ID                TESTS                     STATUS   EXIT CODE   MESSAGE
test-25324770     test,single-path          PASSED   0
test-2886892866   test,subscribe            PASSED   0
test-3109317976   suite,integration-tests   PASSED   0
```

To get the logs from a specific test, use `onit get logs` with the test ID:

```bash
> onit get logs test-2886892866
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
ID                             STATUS
onos-config-569c7d8546-jscg8   RUNNING
```

To get logs for the above node, run the following command:
```bash
> onit get logs onos-config-569c7d8546-jscg8
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
```
To get logs for the above partions, run the following command:
```bash
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
```
To get logs for one of the above histories, run the following command:
```bash
> onit get logs 71a0623c-977c-11e9-8478-acde48001122
=== RUN   test-single-path-test
--- PASS: test-single-path-test (0.04s)
PASS
```

To download logs from a node, you can run `onit fetch logs` command. For example, to download logs from *onos-config-66d54956f5-xwpsh* node, run the following command:
```bash
onit fetch logs onos-config-66d54956f5-xwpsh
```

You can refer to [Debug onos-config in Onit Using Delve](debugging.md) to learn more about debugging of onos-config pod using [*Delve*](https://github.com/go-delve/delve) debugger.


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
> onit get tests
my-test
...
```

The test framework also provides the capability of adding your test to a suite defined in `suites.go`.
To see the suites you can execute:
```bash
> onit get suites
SUITE               TESTS
alltests            single-path, subscribe, transaction
sometests           subscribe, transaction
integration-tests   single-path
```

To add your test to a suite in the init function the register method must be called with the suites parameter:
```go
func init() {
    Registry.RegisterTest("my-test", MyTest, []*runner.TestSuite{AllTests})
}
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
[kind-install]: https://github.com/kubernetes-sigs/kind#installation-and-usage
[MicroK8s]: https://microk8s.io/
[Docker]: https://www.docker.com/
[Atomix]: https://atomix.io
[simulators]: https://github.com/onosproject/simulators
[stratum]: https://www.opennetworking.org/stratum/
