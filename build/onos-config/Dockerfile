ARG ONOS_BUILD_VERSION=stable

FROM onosproject/golang-build:$ONOS_BUILD_VERSION as builder
COPY . /go/src/github.com/onosproject/onos-config
RUN cd /go/src/github.com/onosproject/onos-config && make build

FROM alpine:3.8

USER nobody

COPY --from=builder /go/src/github.com/onosproject/onos-config/build/_output/onos-config /usr/local/bin/onos-config

ENTRYPOINT ["onos-config"]
