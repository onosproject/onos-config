ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION
ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build-plugins

COPY vendor /go/src/github.com/onosproject/onos-config/vendor/
COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-config/
COPY modelplugin /go/src/github.com/onosproject/onos-config/modelplugin/

RUN cd /go/src/github.com/onosproject/onos-config && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}
