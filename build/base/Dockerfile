ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION
ENV GO111MODULE=on
COPY go.mod go.sum /go/src/github.com/onosproject/onos-config/
RUN  go mod download
COPY . /go/src/github.com/onosproject/onos-config
RUN cd /go/src/github.com/onosproject/onos-config && make build
