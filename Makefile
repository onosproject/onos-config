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

local-protos: # @HELP Copies a local version of the onos-api dependency into the vendor directory
ifdef LOCAL_PROTOS
	rm -rf vendor/github.com/onosproject/onos-api/go
	mkdir -p vendor/github.com/onosproject/onos-api/go
	cp -r ${LOCAL_PROTOS}/go/* vendor/github.com/onosproject/onos-api/go
	rm -rf vendor/github.com/onosproject/onos-api/go/vendor
endif

build: # @HELP build the Go binaries and run all validations (default)
build: mod-update local-protos
	go build -mod=vendor -o build/_output/onos-config ./cmd/onos-config

test: # @HELP run the unit tests and source code validation producing a golang style report
test: mod-lint build license_check linters
	go test -race github.com/onosproject/onos-config/...

jenkins-test: # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: mod-lint build license_check linters
	TEST_PACKAGES=github.com/onosproject/onos-config/... ./build/build-tools/build/jenkins/make-unit

helmit-config: integration-test-namespace # @HELP run helmit gnmi tests locally
	helmit test -n test ./cmd/onos-config-tests --suite config

helmit-cli: integration-test-namespace # @HELP run helmit cli tests locally
	helmit test -n test ./cmd/onos-config-tests --suite cli

helmit-rbac: integration-test-namespace # @HELP run helmit gnmi tests locally
	helmit test -n test ./cmd/onos-config-tests --suite rbac --secret keycloak-password=${keycloak_password}

integration-tests: helmit-config helmit-cli helmit-rbac # @HELP run helmit integration tests locally

onos-config-docker: mod-update local-protos # @HELP build onos-config base Docker image
	docker build . -f build/onos-config/Dockerfile \
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


