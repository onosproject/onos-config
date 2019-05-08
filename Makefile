export CGO_ENABLED=0

.PHONY: build

all: image

image: check-version
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config onosproject/onos-config-build:0.3 build
	docker build . -f build/config-manager/Dockerfile -t onosproject/onos-config:${ONOS_CONFIG_VERSION}

check-version:
ifndef ONOS_CONFIG_VERSION
	$(error ONOS_CONFIG_VERSION is undefined)
endif

deps:
	dep ensure -v

lint:
	golint -set_exit_status github.com/onosproject/onos-config/pkg/...
	golint -set_exit_status github.com/onosproject/onos-config/cmd/...

vet:
	go vet github.com/onosproject/onos-config/pkg/...
	go vet github.com/onosproject/onos-config/cmd/...

protos:
	./build/dev-docker/compile-protos.sh

build: test
	export GOOS=linux
	export GOARCH=amd64
	go build -o build/_output/onos-config-manager ./cmd/onos-config-manager

test: protos deps lint vet
	go test github.com/onosproject/onos-config/pkg/...
	go test github.com/onosproject/onos-config/cmd/...

run: deps
	go run cmd/onos-config-manager/config-manager.go