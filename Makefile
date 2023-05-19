# SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

SHELL = bash -e -o pipefail

export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_VERSION ?= latest

build-tools:=$(shell if [ ! -d "./build/build-tools" ]; then cd build && git clone https://github.com/onosproject/build-tools.git; fi)
include ./build/build-tools/make/onf-common.mk

mod-update: # @HELP Download the dependencies to the vendor folder
	go mod tidy
	go mod vendor

mod-lint: mod-update # @HELP ensure that the required dependencies are in place
	# dependencies are vendored, but not committed, go.sum is the only thing we need to check
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

local-deps: # @HELP imports local deps in the vendor folder
local-deps: local-helmit local-onos-api local-onos-lib-go local-onos-ric-sdk-go local-onos-test local-onos-topo

build: # @HELP build the Go binaries and run all validations (default)
build: mod-update local-deps
	go build -mod=vendor -o build/_output/onos-config ./cmd/onos-config

test: # @HELP run the unit tests and source code validation producing a golang style report
test: mod-lint build linters license
	go test -race github.com/onosproject/onos-config/...

jenkins-test: # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: mod-lint build linters license
	TEST_PACKAGES=github.com/onosproject/onos-config/... ./build/build-tools/build/jenkins/make-unit

helmit-config: integration-test-namespace # @HELP run helmit gnmi tests locally
	make helmit-config -C test

helmit-rbac: integration-test-namespace # @HELP run helmit gnmi tests locally
	make helmit-rbac -C test

integration-tests: helmit-config helmit-rbac # @HELP run helmit integration tests locally

onos-config-docker: mod-update local-deps # @HELP build onos-config base Docker image
	docker build --platform linux/amd64 . -f build/onos-config/Dockerfile \
		-t ${DOCKER_REPOSITORY}onos-config:${ONOS_CONFIG_VERSION}

images: # @HELP build all Docker images
images: onos-config-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images kind-only

kind-only: # @HELP deploy the image without rebuilding first
kind-only:
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${DOCKER_REPOSITORY}onos-config:${ONOS_CONFIG_VERSION}

all: build images

publish: # @HELP publish version on github and dockerhub
	./build/build-tools/publish-version ${VERSION} onosproject/onos-config

jenkins-publish: jenkins-tools # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images
	./build/build-tools/release-merge-commit
	./build/build-tools/build/docs/push-docs

clean:: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-config/onos-config ./cmd/onos/onos
	go clean -testcache github.com/onosproject/onos-config/...

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
