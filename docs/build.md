# Building onos-config

> Steps outlined in this page assume that the [development prerequisites](prerequisites.md) are met. 

## Building Locally

The project provides a `Makefile` that supports all the  required to unit test,
validate and build the project.
The makefile supports the following targets:
* `test` - pulls dependencies and runs tests, including lint, vet and license
* `build` - runs `test` and builds an executable
* `all` - runs `test`, `build`, and creates the run-time `onosproject/onos-config` Docker image
  * Requires a local Docker daemon

## Building via Docker Developer Image

Since the build requires a number of tools for depdency generation, linting, vetting, and 
compiling, the project provides a published Docker image `onosproject/onos-config-build`, which 
includes all the required facilities to allow developers to build the project with minimum 
requirements placed on their own machines.
 
To build with the build image, mount the onos-config root to the container
and pass the desired argument to the Makefile entrypoint of the developer image as follows:

```
docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config onosproject/onos-config-build build
```

## Compiling Go Locally

To build Go code locally, developers can use the Go language tools as usual using 
`go build ...` to compile, or `go run ...` to compile and run or `go get ...` to compile and install 
various packages and command executables.

Please see Go language documentation for more information about these tools.

## Executing Unit Tests Locally

To run unit tests locally, run the following:
```bash
go test -v github.com/onosproject/onos-config/cmd/... && \
go test -v github.com/onosproject/onos-config/pkg/...
```

## Bulding Documentation
> Documentation is not published at this time. Soon.

Run locally
```bash
godoc -goroot=$HOME/go
``` 

and then browse at [http://localhost:6060/pkg/github.com/onosproject/onos-config/](http://localhost:6060/pkg/github.com/onosproject/onos-config/)
