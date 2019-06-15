ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION as builder
COPY . /go/src/github.com/onosproject/onos-config
RUN cd /go/src/github.com/onosproject/onos-config && GOOS=linux GOARCH=amd64 make build

FROM alpine:3.9
RUN apk add libc6-compat

USER nobody

COPY --from=builder /go/src/github.com/onosproject/onos-config/build/_output/onos-config /usr/local/bin/onos-config
COPY --from=builder /go/src/github.com/onosproject/onos-config/build/_output/testdevice.so.1.0.0 /usr/local/lib/testdevice.so.1.0.0
COPY --from=builder /go/src/github.com/onosproject/onos-config/build/_output/testdevice.so.2.0.0 /usr/local/lib/testdevice.so.2.0.0
COPY --from=builder /go/src/github.com/onosproject/onos-config/build/_output/devicesim.so.1.0.0 /usr/local/lib/devicesim.so.1.0.0

ENTRYPOINT ["onos-config"]
