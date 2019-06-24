# Testing onos-config

The onos-config project provides an 
[integration testing framework](https://github.com/onosproject/onos-config/pull/374) for
running end-to-end tests on [Kubernetes]. This document describes the steps to creating
and running tests using the integration test framework.

## Setup

To use the integration test framework, it's recommended that you use a local cluster like
[Minikube] or [kind]. To run integration tests, you must first build the integration-tests
image and ensure it's available within the Kubernetes cluster on which you intend to run
tests.


For Minikube, ensure you build the image in the Minikube Docker context:
```bash
> eval $(minikube docker-env)
> make images
```

For kind, you must manually `load` the `onos-config` image and the integration tests image 
into the Kubernetes cluster each time the tests are modified. The `make kind` target will
build and load the onos-config and onos-config-integration-tests images into kind:

```bash
> make kind
```

## Usage

The test framework provides an `onit` command used for running tests in Kubernetes.
To install the `onit` command, use `go get`:

```bash
> go get github.com/onosproject/onos-config/test/cmd/onit
```

The `onit` command provides a set of commands for managing test clusters in Kubernetes,
adding and removing [device simulators][simulators], and running tests on the cluster.

The first step to running tests is to setup a test cluster with `onit setup cluster`:

```bash
> onit setup cluster
I0624 15:36:33.569276   65347 controller.go:100] Setting up test cluster 84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:33.569309   65347 controller.go:175] Setting up test namespace onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:33.592048   65347 controller.go:187] Setting up Atomix controller atomix-controller/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:33.720107   65347 controller.go:210] Waiting for Atomix controller atomix-controller/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941 to become ready
I0624 15:36:47.272478   65347 controller.go:527] Setting up partitions raft/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941
I0624 15:36:47.287600   65347 controller.go:532] Waiting for partitions raft/onos-cluster-84b9795a-96d0-11e9-bd2a-784f43889941 to become ready
```

When a cluster is setup, onit creates a unique namespace within which to create test
resources and then deploys:
* Atomix Kubernetes controller
* Atomix Raft partitions
* onos-config nodes

Once the cluster is setup, the `onit` CLI will be configured to execute future commands against
the test cluster. The current onit cluster context can be seen by running `onit get cluster`:

```bash
> onit get cluster
84b9795a-96d0-11e9-bd2a-784f43889941
```

A list of deployed clusters can be retrieved via `onit get clusters`:

```bash
> onit get clusters
84b9795a-96d0-11e9-bd2a-784f43889941
```

Once a cluster has been setup, you can add simulators to the cluster with `onit add simulator`:

```bash
> onit add simulator my-simulator
```

The `add simulator` command takes an optional name and configuration for the simulator. Once onit
deploys the simulator, it will dynamically reconfigure and redeploy onos-config to connect to
the simulator.

Once the test cluster is setup, to run a test simply run `onit run`:

```bash
> onit run my-test
...
```

When the test is run, the test controller will run the specified tests on the current cluster:

```go
> onit run my-test my-other-test
I0620 12:01:34.503488   14783 controller.go:1022] Starting test job onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941
I0620 12:01:34.519532   14783 controller.go:1090] Waiting for test job onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941 to become ready
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

## Architecture

The integration test framework provides an `onit` command for controlling and running tests.
When `onit` is invoked, the command creates a test `Controller` to run the test.

The default implementation of the test controller is the `kubeController`. The `kubeController`
loads the Kubernetes client from the local context and sets up the test environment:
* Creates a test namespace using a UUID
* Deploys the [Atomix Kubernetes Controller](https://github.com/atomix/atomix-k8s-controller)
* Creates a Raft `PartitionSet`
* Configures and deploys the [simulators](https://github.com/onosproject/simulators) specified by the `--config`
* Configures and deploys `-n` onos-config nodes
* Creates a test `Job` running the `onosproject/onos-config-integration-tests:latest` image
* Streams output from the test `Job` to stdout
* Once tests are complete, fetches the exit code from the test `Job`, cleans up test resources,
and exits

When the `onos-config-integration-tests` image is run, the selected tests are passed to the
container arguments, and tests are run from the test registry by name. The same APIs as are
used by `go test` are used to run integration tests.

[Kubernetes]: https://kubernetes.io
[Minikube]: https://kubernetes.io/docs/setup/learning-environment/minikube/
[kind]: https://github.com/kubernetes-sigs/kind
[simulators]: https://github.com/onosproject/simulators
