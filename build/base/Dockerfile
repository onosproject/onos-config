ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION

RUN go get -u github.com/go-delve/delve/cmd/dlv

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-config/
COPY api/ /go/src/github.com/onosproject/onos-config/api/
COPY cmd/ /go/src/github.com/onosproject/onos-config/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-config/pkg/
COPY vendor/ /go/src/github.com/onosproject/onos-config/vendor/

RUN cd /go/src/github.com/onosproject/onos-config && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}
