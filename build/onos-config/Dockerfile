ARG ONOS_BUILD_VERSION=undefined

FROM onosproject/golang-build:$ONOS_BUILD_VERSION as base

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /build/

WORKDIR /build

RUN go mod download -x

COPY cmd/ /build/cmd/
COPY pkg/ /build/pkg/

RUN cd /build && make ${ONOS_MAKE_TARGET}

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=base /build/build/_output/onos-config /usr/local/bin/onos-config

ENTRYPOINT ["onos-config"]
