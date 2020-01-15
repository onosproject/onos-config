FROM alpine:3.8

RUN apk upgrade --update --no-cache && apk add libc6-compat

USER nobody

ADD build/onos-config-benchmarks/_output/bin/onos-config-benchmarks /usr/local/bin/onos-config-benchmarks

ENTRYPOINT ["onos-config-benchmarks"]