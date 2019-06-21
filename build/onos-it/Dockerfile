FROM golang:1.12

ENV GO111MODULE=on
COPY . /go/src/github.com/onosproject/onos-config
WORKDIR /go/src/github.com/onosproject/onos-config
RUN cd /go/src/github.com/onosproject/onos-config && go build ./...

ENTRYPOINT ["go", "run", "github.com/onosproject/onos-config/test/cmd/onit", "test"]