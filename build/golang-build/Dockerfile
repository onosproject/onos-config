FROM golang:1.12

COPY dep-*.sha256 .

RUN apt-get update && \
    mkdir release && \
    wget -q -O release/dep-$(go env GOOS)-$(go env GOARCH) https://github.com/golang/dep/releases/download/v0.5.0/dep-$(go env GOOS)-$(go env GOARCH) && \
    sha256sum -c dep-$(go env GOOS)-$(go env GOARCH).sha256 && \
    mv release/dep-$(go env GOOS)-$(go env GOARCH) /go/bin/dep && \
    chmod +x /go/bin/dep && \
    rmdir release && \
    rm dep-*.sha256 && \
    apt-get install -y protobuf-compiler && \
    go get -u github.com/golang/protobuf/protoc-gen-go && \
    mkdir -p /go/src/github.com/google && \
    git clone --branch master https://github.com/google/protobuf /go/src/github.com/google/protobuf && \
    go get -u golang.org/x/lint/golint && \
    mkdir -p /go/src/github.com/

WORKDIR "/go/src/github.com"

ENTRYPOINT ["make"]
