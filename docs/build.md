<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# Building onos-config
In order the build the project, developers are expected to install the 
required [development tools](../../developers/prerequisites.md). 

Currently, the project build and validation is driven by a top-level `Makefile`, which supports the following usage:
```bash
> make help
Makefile:build                  build the Go binaries and run all validations (default)
Makefile:clean:                 remove all the build artifacts
Makefile:images                 build all Docker images
Makefile:jenkins-test           run the unit tests and source code validation producing a junit style report for Jenkins
Makefile:kind-only              deploy the image without rebuilding first
Makefile:kind                   build Docker images and add them to the currently configured kind cluster
Makefile:local-deps             imports local deps in the vendor folder
Makefile:local-helmit           Copies a local version of the helmit dependency into the vendor directory
Makefile:local-onos-api         Copies a local version of the onos-api dependency into the vendor directory
Makefile:local-onos-lib-go      Copies a local version of the onos-lib-go dependency into the vendor directory
Makefile:local-onos-ric-sdk-go  Copies a local version of the onos-ric-sdk-go dependency into the vendor directory
Makefile:local-onos-test        Copies a local version of the onos-test dependency into the vendor directory
Makefile:local-onos-topo        Copies a local version of the onos-topo dependency into the vendor directory
Makefile:mod-update             Download the dependencies to the vendor folder
Makefile:publish                publish version on github and dockerhub
Makefile:test                   run the unit tests and source code validation producing a golang style report
```

## Building Go binaries
To build the project, simply type `make`. This will check for required dependencies, compile the Go binaries 
and then perform all required validation steps, which includes unit tests, Go code formating, Go lint, Go vetting
and license header compliance check. In future, there may be other tests.

| Note that since the build relies on Go modules, you must `export GO111MODULE=on`.
## Building Docker images
To allow deployment of onos-config in a Kubernetes cluster, the `Makefile` allows creation of two separate Docker 
images.

The main Docker image is `onosproject/onos-config`, which is the main program that acts as a server that provides 
various gRPC interfaces to application. This include `gNMI` and the `AdminService` and `DiagnosticService`. The
second Docker image is `onosproject/onos-cli`, which provides a command-line shell that can be deployed as an
ephemeral container inside the Kubernetes cluster and which provides access to the `onos` CLI commands for 
remotely interacting with the services provided by `onosproject/onos-config`.

You can build both images by running `make images`.

## Compiling protobufs
To compile Google Protocol Buffer files (`*.proto`) and to generate Go source files from them, simply run
`make protos`. Provided you changed the source `*.proto` files, this will modify the corresponding `*.pb.go` source
files. Although these files are auto-generated, developers are expected to check them in, anytime they change as
a result of changing the `*.proto` files.

> The protoc compiler is run using `onosproject/proto-go` Docker image, which has been published to remove the
need for developers to install their own protoc compiler and its Go plugin. The Makefile makes this transparent.


## Bulding Documentation
Documentation is published at [https://godoc.org/github.com/onosproject/onos-config](https://godoc.org/github.com/onosproject/onos-config)

If you wish to see a version of it locally run:
```bash
godoc -goroot=$HOME/go
``` 

and then browse at [http://localhost:6060/pkg/github.com/onosproject/onos-config/](http://localhost:6060/pkg/github.com/onosproject/onos-config/)
