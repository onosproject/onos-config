FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/onos-config-manager /usr/local/bin/onos-config-manager

ENTRYPOINT ["onos-config-manager"]
