FROM golang:1.16-alpine3.13 AS build

RUN apk upgrade --update --no-cache && apk add --update make gcc musl-dev git

COPY go.mod go.sum /build/

WORKDIR /build

RUN go mod download -x

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile /build/
COPY cmd/ /build/cmd/
COPY pkg/ /build/pkg/
COPY model/ /build/model/

RUN make ${ONOS_MAKE_TARGET}

FROM alpine:3.13
RUN apk add libc6-compat

USER nobody

COPY --from=build /build/build/_output/onos-config /usr/local/bin/onos-config

ENTRYPOINT ["onos-config"]
