ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION

RUN go get -u github.com/go-delve/delve/cmd/dlv

ENV GO111MODULE=on

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-config/
COPY cmd/ /go/src/github.com/onosproject/onos-config/cmd/
COPY configs/ /go/src/github.com/onosproject/onos-config/configs/
COPY pkg/ /go/src/github.com/onosproject/onos-config/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-config/vendor/

RUN cd /go/src/github.com/onosproject/onos-config && GOFLAGS=-mod=vendor make build
