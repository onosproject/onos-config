FROM golang:1.16-alpine3.13

RUN apk upgrade --update --no-cache && apk add --update make gcc musl-dev git

COPY go.mod go.sum /onos-config/

WORKDIR /onos-config

RUN go mod download -x

COPY Makefile /onos-config/
COPY cmd/ /onos-config/cmd/
COPY pkg/ /onos-config/pkg/
COPY model/ /onos-config/model/

RUN go run github.com/onosproject/onos-config/cmd/config-model init
