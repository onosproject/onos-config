FROM onosproject/golang-build:v0.6.3 AS build

ENV GO111MODULE=on
ARG ONOS_MAKE_TARGET=build

COPY Makefile go.mod go.sum /go/src/github.com/onosproject/onos-config/
WORKDIR /go/src/github.com/onosproject/onos-config
RUN go mod download -x

COPY Makefile /go/src/github.com/onosproject/onos-config/
COPY cmd/ /go/src/github.com/onosproject/onos-config/cmd/
COPY pkg/ /go/src/github.com/onosproject/onos-config/pkg/

RUN make ${ONOS_MAKE_TARGET}

FROM alpine:3.11
RUN apk add libc6-compat

USER nobody

COPY --from=build /go/src/github.com/onosproject/onos-config/build/_output/onos-config /usr/local/bin/onos-config

ENTRYPOINT ["onos-config"]
