# SPDX-License-Identifier: Apache-2.0
# Copyright 2019 Open Networking Foundation
# Copyright 2024 Intel Corporation

SHELL = bash -e -o pipefail

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_VERSION ?= latest

GOLANG_CI_VERSION := v1.52.2

all: build docker-build

local-deps: # @HELP imports local deps in the vendor folder
local-deps: local-helmit local-onos-api local-onos-lib-go local-onos-ric-sdk-go local-onos-test local-onos-topo

mod-deps: # @HELP update local dependency in go.mod and go.sum
	go mod tidy
	go mod vendor

build: # @HELP build the Go binaries and run all validations (default)
build: mod-deps local-deps
	go build -mod=vendor -o build/_output/onos-config ./cmd/onos-config

test: # @HELP run the unit tests and source code validation producing a golang style report
test: build lint license
	go test -race github.com/onosproject/onos-config/...

docker-build-onos-config: # @HELP build onos-config base Docker image
docker-build-onos-config: local-deps
	docker build --platform linux/amd64 . -f build/onos-config/Dockerfile \
		-t onosproject/onos-config:${ONOS_CONFIG_VERSION}

docker-build: # @HELP build all Docker images
docker-build: build docker-build-onos-config

docker-push-onos-config: # @HELP push onos-pci Docker image
	docker push onosproject/onos-config:${ONOS_CONFIG_VERSION}

docker-push: # @HELP push docker images
docker-push: docker-push-onos-config

lint: # @HELP examines Go source code and reports coding problems
	golangci-lint --version | grep $(GOLANG_CI_VERSION) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b `go env GOPATH`/bin $(GOLANG_CI_VERSION)
	golangci-lint run --timeout 15m

license: # @HELP run license checks
	rm -rf venv
	python3 -m venv venv
	. ./venv/bin/activate;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install reuse;\
	reuse lint

check-version: # @HELP check version is duplicated
	./build/bin/version_check.sh all

clean:: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-config/onos-config ./cmd/onos/onos
	go clean github.com/onosproject/onos-config/...

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '

local-helmit: # @HELP Copies a local version of the helmit dependency into the vendor directory
ifdef LOCAL_HELMIT
	rm -rf vendor/github.com/onosproject/helmit/go
	mkdir -p vendor/github.com/onosproject/helmit/go
	cp -r ${LOCAL_HELMIT}/go/* vendor/github.com/onosproject/helmit/go
endif

local-onos-api: # @HELP Copies a local version of the onos-api dependency into the vendor directory
ifdef LOCAL_ONOS_API
	rm -rf vendor/github.com/onosproject/onos-api/go
	mkdir -p vendor/github.com/onosproject/onos-api/go
	cp -r ${LOCAL_ONOS_API}/go/* vendor/github.com/onosproject/onos-api/go
endif

local-onos-lib-go: # @HELP Copies a local version of the onos-lib-go dependency into the vendor directory
ifdef LOCAL_ONOS_LIB_GO
	rm -rf vendor/github.com/onosproject/onos-lib-go
	mkdir -p vendor/github.com/onosproject/onos-lib-go
	cp -r ${LOCAL_ONOS_LIB_GO}/* vendor/github.com/onosproject/onos-lib-go
endif

local-onos-ric-sdk-go: # @HELP Copies a local version of the onos-ric-sdk-go dependency into the vendor directory
ifdef LOCAL_ONOS_RIC_SDK
	rm -rf vendor/github.com/onosproject/onos-ric-sdk-go
	mkdir -p vendor/github.com/onosproject/onos-ric-sdk-go
	cp -r ${LOCAL_ONOS_RIC_SDK}/* vendor/github.com/onosproject/onos-ric-sdk-go
endif

local-onos-topo: # @HELP Copies a local version of the onos-topo dependency into the vendor directory
ifdef LOCAL_ONOS_TOPO
	rm -rf vendor/github.com/onosproject/onos-topo
	mkdir -p vendor/github.com/onosproject/onos-topo
	cp -r ${LOCAL_ONOS_TOPO}/* vendor/github.com/onosproject/onos-topo
endif

local-onos-test: # @HELP Copies a local version of the onos-test dependency into the vendor directory
ifdef LOCAL_ONOS_TEST
	rm -rf vendor/github.com/onosproject/onos-test
	mkdir -p vendor/github.com/onosproject/onos-test
	cp -r ${LOCAL_ONOS_TEST}/* vendor/github.com/onosproject/onos-test
endif
