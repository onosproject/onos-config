ARG ONOS_CONFIG_BASE_VERSION=latest

# The base image contains the onos-config binaries
FROM onosproject/onos-config-base:$ONOS_CONFIG_BASE_VERSION as base

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/onos-config /usr/local/bin/onos-config

ENTRYPOINT ["onos-config"]
