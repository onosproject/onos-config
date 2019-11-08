FROM golang:1.12

RUN apt-get update && \
    apt-get install unzip

RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.9.1/protoc-3.9.1-linux-x86_64.zip && \
    unzip -o protoc-3.9.1-linux-x86_64.zip -d /usr/local bin/protoc && \
    unzip -o protoc-3.9.1-linux-x86_64.zip -d /usr/local include/* && \
    rm -rf protoc-3.9.1-linux-x86_64.zip

RUN go get -u github.com/golang/protobuf/protoc-gen-go && \
    go get -u github.com/gogo/protobuf/proto && \
    go get -u github.com/gogo/protobuf/gogoproto && \
    go get -u github.com/gogo/protobuf/protoc-gen-gofast && \
    go get -u github.com/gogo/protobuf/protoc-gen-gogo && \
    go get -u github.com/gogo/protobuf/protoc-gen-gogofast && \
    go get -u github.com/gogo/protobuf/protoc-gen-gogofaster && \
    go get -u github.com/gogo/protobuf/protoc-gen-gogoslick && \
    go get -u github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc

RUN mkdir -p /go/src/github.com/google && \
    git clone --branch master https://github.com/google/protobuf /go/src/github.com/google/protobuf && \
    git clone --branch master https://github.com/openconfig/gnmi /go/src/github.com/openconfig/gnmi && \
    mkdir -p /go/src/github.com/ &&\
    wget "https://github.com/grpc/grpc-web/releases/download/1.0.5/protoc-gen-grpc-web-1.0.5-linux-x86_64" --quiet && \
    mv protoc-gen-grpc-web-1.0.5-linux-x86_64 /usr/local/bin/protoc-gen-grpc-web && \
    chmod +x /usr/local/bin/protoc-gen-grpc-web

WORKDIR "/go/src/github.com/"

ENTRYPOINT ["/bin/bash"]
