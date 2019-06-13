# Developer Docker Container
The purpose of this docker container is to provide a minimal environment for building and
validating the project. As such this container can be used as part of the Travis CI.

## Using the Container
The container can be used by running it with a mount that points to the local workspace and setting the 
project top-level directory as the work directory. It will run make using the top-level `Makefile` in 
that work directory.
Optionally, you may specify the desired make target if you don't wish to run the default one.

For example, to build onos-config project, run the following:

```bash
> docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config \
    -w /go/src/github.com/onosproject/onos-config \
    onosproject/golang-build:stable
```

## Building the Container
If you need to customize and rebuild the container, run the following command:

```bash
> docker build -t onosproject/golang-build:latest build/golang-build
```

Note that to use the locally built Docker image, you will have to use the `latest` 
tag instead of the `stable` one.