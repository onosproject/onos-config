ARG ONOS_CONFIG_BASE_VERSION=latest

FROM onosproject/onos-config-base:$ONOS_CONFIG_BASE_VERSION as base

FROM golang:1.12.6-alpine3.9 as debugBuilder

RUN apk upgrade --update --no-cache && apk add git && \
    go get -u github.com/go-delve/delve/cmd/dlv && \
    go get -u github.com/openconfig/gnmi/cmd/gnmi_cli

FROM alpine:3.9

RUN apk upgrade --update --no-cache && apk add bash bash-completion libc6-compat

COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/onos-config-debug /usr/local/bin/onos-config
COPY --from=base /go/src/github.com/onosproject/onos-config/build/_output/*-debug.so.* /usr/local/lib/
COPY --from=debugBuilder /go/bin/dlv /usr/local/bin/dlv
COPY --from=debugBuilder /go/bin/gnmi_cli /usr/local/bin/gnmi_cli

RUN echo "#!/bin/sh" >> /usr/local/bin/onos-config-debug && \
    echo "dlv --listen=:40000 --headless=true --accept-multiclient=true --continue --api-version=2 --log exec /usr/local/bin/onos-config -- \"\$@\"" >> /usr/local/bin/onos-config-debug && \
    chmod +x /usr/local/bin/onos-config-debug

RUN addgroup -S onos-config && adduser -S -G onos-config onos-config
USER onos-config
WORKDIR /home/onos-config

ENV ATOMIX_CONTROLLER=atomix-controller.kube-system.svc.cluster.local:5679
ENV ATOMIX_NAMESPACE=default

RUN cp /etc/profile /home/onos-config/.bashrc

ENTRYPOINT ["onos-config-debug"]
