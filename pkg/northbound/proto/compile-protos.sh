#!/bin/sh

proto_imports=".:${GOPATH}/src/github.com/google/protobuf/src:${GOPATH}/src"

protoc -I=$proto_imports --go_out=plugins=grpc:. pkg/northbound/proto/*.proto
