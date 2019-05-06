# Developer Docker Container
The purpose of this docker container is to provide a minimal environment for building and
validating the project. As such this container can be used as part of the Travis CI.

## Building the Container
To build the container run the following command:

```sh
docker build -t onos-config:0.1 build/dev-docker
```

## Using the Container
The container can be used by running it with a mount that points to the local workspace and run the _make_
on the top-level _Makefile_ on the project. The _Makefile_ will use the go-dep tool to download
the project dependencies and then will build the _onos-config_ project.

This can be accomplished by running the following command:

```sh
docker run -it -v `pwd`:/go/src/github.com/opennetworkinglab/onos-config onos-config:0.1 build
```

