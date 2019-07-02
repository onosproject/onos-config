ARG ONOS_CONFIG_BASE_VERSION=latest

FROM onosproject/onos-config-base:$ONOS_CONFIG_BASE_VERSION as base

FROM alpine:3.8

COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/onit-k8s /usr/local/bin/onit-k8s

USER nobody

ENTRYPOINT ["onit-k8s"]