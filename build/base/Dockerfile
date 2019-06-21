ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION
COPY . /go/src/github.com/onosproject/onos-config
RUN cd /go/src/github.com/onosproject/onos-config && make build
