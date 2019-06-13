# Protoc Go Docker Container
The purpose of this docker container is to allow running the protobuf compiler in an 
isolated environment to free the developers from having to install protoc themselves.

## Using the Container
The container can be used by running it with a mount that points to the local workspace,
set the project-specific work directory and run the project-specific `compile-protos.sh` 
script as an entrypoint.

For example, to compile the onos-config project proto files, run the following:

```bash
> docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config \
    -w /go/src/github.com/onosproject/onos-config \
    --entrypoint pkg/northbound/proto/compile-protos.sh \
    onosproject/protoc-go:stable 
```

## Building the Container
If you need to customize and rebuild the container, run the following command:

```bash
> docker build -t onosproject/protoc-go:latest build/protoc-go
```

Note that to use the locally built Docker image, you will have to use the `latest` 
tag instead of the `stable` one.
