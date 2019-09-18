ARG ONOS_CONFIG_BASE_VERSION=latest
ARG ONOS_CONFIG_PLUGINS_VERSION=latest

# The base image contains the onos-config binaries
FROM onosproject/onos-config-base:$ONOS_CONFIG_BASE_VERSION as base

# The plugins image contains plugin binaries
FROM onosproject/onos-config-plugins:$ONOS_CONFIG_PLUGINS_VERSION as plugins

FROM alpine:3.9
RUN apk add libc6-compat

USER nobody

COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/onos-config /usr/local/bin/onos-config
COPY --from=plugins /go/src/github.com/onosproject/onos-config/build/_output/*.so.* /usr/local/lib/

ENTRYPOINT ["onos-config"]
