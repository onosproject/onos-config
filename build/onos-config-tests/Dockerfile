FROM alpine:3.8

RUN apk upgrade --update --no-cache && apk add libc6-compat

USER nobody

ADD build/onos-config-tests/_output/bin/onos-config-tests /usr/local/bin/onos-config-tests

ENTRYPOINT ["onos-config-tests"]
