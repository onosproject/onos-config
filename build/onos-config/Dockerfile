FROM onosproject/config-model-build:v1.0 AS build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /build/

COPY Makefile /build/
COPY cmd/ /build/cmd/
COPY pkg/ /build/pkg/
COPY vendor/ /build/vendor/

RUN make ${ONOS_MAKE_TARGET}

FROM alpine:3.13
RUN apk add libc6-compat

USER nobody

COPY --from=build /build/build/_output/onos-config /usr/local/bin/onos-config

ENTRYPOINT ["onos-config"]
