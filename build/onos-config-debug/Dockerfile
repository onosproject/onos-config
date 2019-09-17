ARG ONOS_CONFIG_BASE_VERSION=latest
ARG ONOS_CONFIG_PLUGINS_VERSION=latest

# The base image contains the onos-config binaries
FROM onosproject/onos-config-base:$ONOS_CONFIG_BASE_VERSION as base

# The plugins image contains plugin binaries
FROM onosproject/onos-config-plugins:$ONOS_CONFIG_PLUGINS_VERSION as plugins

FROM ubuntu:18.04

COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/onos-config-debug /usr/local/bin/onos-config
COPY --from=base /go/bin/dlv /usr/local/bin/dlv
COPY --from=plugins /go/src/github.com/onosproject/onos-config/build/_output/*-debug.so.* /usr/local/lib/

RUN echo "#!/bin/sh" >> /usr/local/bin/onos-config-debug && \
    echo "dlv --listen=:40000 --headless=true --accept-multiclient=true --continue --api-version=2 --log exec /usr/local/bin/onos-config -- \"\$@\"" >> /usr/local/bin/onos-config-debug && \
    chmod +x /usr/local/bin/onos-config-debug

RUN adduser onos-config
USER onos-config
WORKDIR /home/onos-config

ENTRYPOINT ["onos-config-debug"]
