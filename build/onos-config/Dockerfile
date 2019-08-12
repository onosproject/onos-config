ARG ONOS_CONFIG_BASE_VERSION=latest

FROM onosproject/onos-config-base:$ONOS_CONFIG_BASE_VERSION as base

FROM alpine:3.9
RUN apk add libc6-compat

USER nobody

COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/onos-config /usr/local/bin/onos-config
COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/*.so.* /usr/local/lib/

ENTRYPOINT ["onos-config"]
