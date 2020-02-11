ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION
ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build-plugins

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-config/
COPY vendor/ /go/src/github.com/onosproject/onos-config/vendor/

# "go mod vendor" will not copy over anything in a "main" package, so we have
# to get them and copy them over in to vendor folder
RUN cd /go/src/github.com/onosproject/onos-config && \
go mod download github.com/onosproject/config-models/modelplugin/devicesim-1.0.0 && \
cp /go/pkg/mod/github.com/onosproject/config-models/modelplugin/devicesim-1.0.0*/modelmain.go vendor/github.com/onosproject/config-models/modelplugin/devicesim-1.0.0 && \
go mod download github.com/onosproject/config-models/modelplugin/testdevice-1.0.0 && \
cp /go/pkg/mod/github.com/onosproject/config-models/modelplugin/testdevice-1.0.0*/modelmain.go vendor/github.com/onosproject/config-models/modelplugin/testdevice-1.0.0 && \
go mod download github.com/onosproject/config-models/modelplugin/testdevice-2.0.0 && \
cp /go/pkg/mod/github.com/onosproject/config-models/modelplugin/testdevice-2.0.0*/modelmain.go vendor/github.com/onosproject/config-models/modelplugin/testdevice-2.0.0 && \
go mod download github.com/onosproject/config-models/modelplugin/stratum-1.0.0 && \
cp /go/pkg/mod/github.com/onosproject/config-models/modelplugin/stratum-1.0.0*/modelmain.go vendor/github.com/onosproject/config-models/modelplugin/stratum-1.0.0 && \
go mod download github.com/onosproject/config-models && \
cp /go/pkg/mod/github.com/onosproject/config-models*/cmd/copylibandstay.go vendor/github.com/onosproject/config-models/cmd

RUN cd /go/src/github.com/onosproject/onos-config && GOFLAGS=-mod=vendor make ${ONOS_MAKE_TARGET}
