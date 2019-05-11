# Development Prerequisites
This document provides an overview of the tools and packages needed to work on onos-config.

## IDE
The core team uses the [GoLand IDE](/https://www.jetbrains.com/go/) from JetBrains, but there are
many other options. 
Microsoft's [Visual Studio Code](/https://code.visualstudio.com) is one such option
and is available as a free download.

## License
The project requires that all Go source files are properly annotated using the Apache 2.0 License.
This requirement is enforced by the CI process. Therefore, it is strongly recommended that developers
setup their IDE to automatically include the [license text](../build/licensing/boilerplate.go.txt)
automatically.

[GoLand IDE can be easily setup to do this](license_goland.md) and other IDEs will have a similar mechanism.


## Docker
While this project can be built entirely on the developer machine, this does require that a number of
developer tools be installed first (list will be available shortlu. To minimize the setup requirements
for the developer's own machine, a Docker image has been published that allows this project to be
build without these dependencies. 

However, to take advantage of this, Docker must be installed locally.
See the [Docker installation guide](https://docs.docker.com/install/) for details.


> Other tools will be added shortly.

