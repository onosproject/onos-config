# Prerequisites for Installing and Running onos-config
This document provides an overview of the tools and packages needed to work on onos-config

## IDE
The core team uses the [GoLand IDE](/https://www.jetbrains.com/go/) from JetBrains, but there are
many other options. 
Microsoft's [Visual Studio Code](/https://code.visualstudio.com) is one such option
and is available as a free download.

## License
Please do setup your IDE to include automatically the APACHE 2.0 License in your new files. 
You can find the licens we expect for .go files [here](../build/licensing/boilerplate.go.txt) 
A [guide](license_goland.md) for golang is provided. 

**Note**   
The CI tests the are run as part of verification of a pull request require license to be in place

## Docker
While this project can be built entirely on the developer machine, this does require that a number of
developer tools be installed first (list will be available shortlu. To minimize the setup requirements
for the developer's own machine, a Docker image has been published that allows this project to be
build without these dependencies. 

However, to take advantage of this, Docker must be installed locally.
See the [Docker installation guide](https://docs.docker.com/install/) for details.

