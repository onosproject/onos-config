# Development Prerequisites
This document provides an overview of the tools and packages needed to work on onos-config.

## IDE
The core team uses the [GoLand IDE](/https://www.jetbrains.com/go/) from JetBrains, but there are
many other options. 
Microsoft's [Visual Studio Code](/https://code.visualstudio.com) is one such option
and is available as a free download.

## License
The project requires that all Go source files are properly annotated using the Apache 2.0 License.
Since this requirement is enforced by the CI process, it is strongly recommended that developers
setup their IDE to include the [license text](../build/licensing/boilerplate.go.txt)
automatically.

[GoLand IDE can be easily setup to do this](license_goland.md) and other IDEs will have a similar mechanism.

## Go Tools
Since the project is authored mainly in the Go programming language, the 
[Go tools](https://golang.org/doc/install) are required to be installed on the developer machine in 
in order to build and execute the code. It is highly recommended that developers do so.

## Docker
While this project can be built entirely on the developer machine, this does require that several
developer tools are installed first - see Additional Tools below. To minimize the setup requirements
for the developer's own machine, a Docker image has been published that has these tools installed and
allows this project to be build without developers having to install the additional tools themselves
 - see [build.md](build.md).

However, to take advantage of this, Docker must be installed locally.
See the [Docker installation guide](https://docs.docker.com/install/) for details.

## Additional Tools
In order to execute the entire build locally (as opposed to using the provided Docker image),
developers will also need to install the following tools:

### Go Lint
Go Lint is required to validate that the Go source code complies with the established style 
guidelines. See https://github.com/golang/lint for more information about the tool and how to 
install it.

### Go Dep
Go dependency manager is required to automatically manage package dependencies.
See https://github.com/golang/dep for more information on how to install it.

### Protobuf Compiler
Required to compile the administrative and diagnostic gRPC interfaces.
See https://github.com/protocolbuffers/protobuf to lean more about protocol buffers 
and how to install the _protobuf compiler_ for different programming languages.

