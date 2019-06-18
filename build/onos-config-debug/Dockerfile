ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION as onosBuilder
COPY . /go/src/github.com/onosproject/onos-config
RUN cd /go/src/github.com/onosproject/onos-config && GOOS=linux GOARCH=amd64 DEBUG=true make build

FROM golang:1.12.6-alpine3.9 as debugBuilder

RUN apk upgrade --update --no-cache && apk add git && go get -u github.com/go-delve/delve/cmd/dlv

FROM alpine:3.9
RUN apk add libc6-compat

COPY --from=onosBuilder /go/src/github.com/onosproject/onos-config/build/_output/onos-config /usr/local/bin/onos-config
COPY --from=onosBuilder /go/src/github.com/onosproject/onos-config/build/_output/testdevice.so.1.0.0 /usr/local/lib/testdevice.so.1.0.0
COPY --from=onosBuilder /go/src/github.com/onosproject/onos-config/build/_output/testdevice.so.2.0.0 /usr/local/lib/testdevice.so.2.0.0
COPY --from=onosBuilder /go/src/github.com/onosproject/onos-config/build/_output/devicesim.so.1.0.0 /usr/local/lib/devicesim.so.1.0.0
COPY --from=debugBuilder /go/bin/dlv /usr/local/bin/dlv

RUN echo "#!/bin/sh" >> /usr/local/bin/onos-config-debug && \
    echo "dlv --listen=:40000 --headless=true --api-version=2 exec /usr/local/bin/onos-config -- \"\$@\"" >> /usr/local/bin/onos-config-debug && \
    chmod +x /usr/local/bin/onos-config-debug

RUN addgroup -S onos-config && adduser -S -G onos-config onos-config
USER onos-config
WORKDIR /home/onos-config

ENTRYPOINT ["onos-config-debug"]
