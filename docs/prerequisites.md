# Development Prerequisites
This document provides an overview of the tools and packages needed to work on and to build onos-config.
Developers are expected to have these tools installed on the machine where the project is built.

## Go Tools
Since the project is authored mainly in the Go programming language, the project requires [Go tools] 
in order to build and execute the code.

## Go Lint
[Go lint] is required to validate that the Go source code complies with the established style 
guidelines.

## Docker
[Docker] is required to build the project Docker images and also to compile `*.proto` files into Go source files.

## Local kubernetes environment
Some form of local kubernetes development environment is also needed.
The core team uses [Kind], but there are other options such as [Minikube].

## IDE
Some form of an integrated development environment is also recommended.
The core team uses the [GoLand IDE] from JetBrains, but there are many other options. 
Microsoft's [Visual Studio Code] is one such option and is available as a free download.

## License
The project requires that all Go source files are properly annotated using the Apache 2.0 License.
Since this requirement is enforced by the CI process, it is strongly recommended that developers
setup their IDE to include the [license text](../build/licensing/boilerplate.go.txt)
automatically.

[GoLand IDE can be easily setup to do this](license_goland.md) and other IDEs will have a similar mechanism.


[Go tools]: https://golang.org/doc/install
[Go lint]: https://github.com/golang/lint
[Go lint]: https://github.com/golang/lint
[Docker]: https://docs.docker.com/install/
[Kind]: https://github.com/kubernetes-sigs/kind
[Minikube]: https://kubernetes.io/docs/tasks/tools/install-minikube/

[GoLand IDE]: /https://www.jetbrains.com/go/
[Visual Studio Code]: /https://code.visualstudio.com