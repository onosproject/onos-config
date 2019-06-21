ARG ONOS_CONFIG_BASE_VERSION=latest

FROM onosproject/onos-config-base:$ONOS_CONFIG_BASE_VERSION as base

FROM alpine:3.8

COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/onit /usr/local/bin/onit

USER nobody

ENTRYPOINT ["onit", "test"]