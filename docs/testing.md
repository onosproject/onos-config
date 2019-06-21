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
into the Kubernetes cluster each time the tests are modified:

```bash
> make images
...
> kind load docker-image onosproject/onos-config:latest
> kind load docker-image onosproject/onos-config-integration-tests:latest
```

## Usage

The test framework provides an `onit` command used for running tests in Kubernetes.
To install the `onit` command, use `go get`:

```bash
> go get github.com/onosproject/onos-config/test/cmd/onit
```

Then, to run a test, simply run `onit run`:

```bash
> onit run my-test
...
```

When the test is run, the test controller will setup a new test cluster in Kubernetes and
run the specified tests:

```go
> onit run my-test my-other-test
I0620 11:59:59.990621   14783 controller.go:147] Setting up test namespace onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941
I0620 12:00:00.011219   14783 controller.go:159] Setting up Atomix controller atomix-controller/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941
I0620 12:00:00.099428   14783 controller.go:176] Waiting for Atomix controller atomix-controller/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941 to become ready
I0620 12:00:12.060342   14783 controller.go:435] Setting up partitions raft/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941
I0620 12:00:12.066366   14783 controller.go:440] Waiting for partitions raft/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941 to become ready
I0620 12:01:20.270180   14783 controller.go:547] Setting up simulator device-simulator-1/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941
I0620 12:01:20.316644   14783 controller.go:554] Waiting for simulator device-simulator-1/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941 to become ready
I0620 12:01:27.679763   14783 controller.go:712] Setting up onos-config cluster onos-config/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941
I0620 12:01:28.296125   14783 controller.go:726] Waiting for onos-config cluster onos-config/onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941 to become ready
I0620 12:01:34.503488   14783 controller.go:1022] Starting test job onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941
I0620 12:01:34.519532   14783 controller.go:1090] Waiting for test job onos-test-9a4c1c3c-938d-11e9-8e49-784f43889941 to become ready
```

By default, the tests will run with a single onos-config node and a single Raft partition. These options
can be overridden by flags to the `run` subcommand:
* `--nodes`/`-n` the number of onos-config nodes to deploy
* `--partitions`/`-p` the number of Raft partitions to create
* `--partitionSize`/`-s` the size of each Raft partition to create
* `--config`/`-c` the store configuration to use

By default, the tests will run with no simulators configured using the sample store configurations
from `/configs`. The store configuration used by the test framework can be overridden by providing
a `--config`. The `--config` flag expects a named configuration pointing to a `.json` file present
in `test/configs`. By default, the `default` configuration is used, which means the configuration
is loaded from `test/configs/default.json`.

Simulators can be added to the test cluster by adding simulator configurations to the `simulators` 
object in the configuration JSON:

```json
{
  ...
  "simulators": {
    "device-1": {
      "openconfig-interfaces:interfaces": {
        "interface": [
          {
            "name": "admin",
            "config": {
              "name": "admin"
            }
          }
        ]
      },
      ...
    }
  }
}
```

The integration test framework will automatically deploy a simulator with each configuration
when setting up the cluster. The key for each device is the device ID, so devices defined in
the configuration can be accessed in gNMI targets by their key.

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
