FROM alpine:3.8

RUN apk upgrade --update --no-cache && apk add git
RUN git clone --branch master https://github.com/onosproject/onos-helm-charts /etc/charts

RUN addgroup -S test && adduser -S test -G test

USER test

ADD _output/bin/onos-config-benchmarks /usr/local/bin/onos-config-benchmarks

ENTRYPOINT ["onos-config-benchmarks"]
