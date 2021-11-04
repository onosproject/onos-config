FROM golang:1.16-alpine3.13 AS build

RUN apk upgrade --update --no-cache && apk add --update make gcc musl-dev git

RUN mkdir -p /build

COPY go.mod go.sum /build/

WORKDIR /build

RUN go mod download -x

ENV GO111MODULE=on

COPY cmd/ /build/cmd/
COPY model/ /build/model/

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config/cmd/config-model-registry", "serve"]
