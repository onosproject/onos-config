FROM golang:1.12

RUN apt-get update && \
    apt-get install -y protobuf-compiler && \
    go get -u github.com/golang/protobuf/protoc-gen-go && \
    mkdir -p /go/src/github.com/google && \
    git clone --branch master https://github.com/google/protobuf /go/src/github.com/google/protobuf && \
    git clone --branch master https://github.com/openconfig/gnmi /go/src/github.com/openconfig/gnmi && \
    mkdir -p /go/src/github.com/

WORKDIR "/go/src/github.com/"

ENTRYPOINT ["/bin/bash"]
