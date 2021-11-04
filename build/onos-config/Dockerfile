ARG ONOS_CONFIG_VERSION=latest
ARG ONOS_MAKE_TARGET=build

FROM onosproject/onos-config-base:${ONOS_CONFIG_VERSION} AS build

RUN make ${ONOS_MAKE_TARGET}

FROM alpine:3.13
RUN apk add libc6-compat

USER nobody

COPY --from=build /onos-config/build/_output/onos-config /usr/local/bin/onos-config
COPY --from=build /onos-config/mod.hash /etc/onos/mod/

ENTRYPOINT ["onos-config"]
