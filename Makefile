export CGO_ENABLED=1
export GO111MODULE=on

.PHONY: build

DOCKER_REPOSITORY ?= onosproject/
KIND_CLUSTER_NAME ?= kind
ONOS_CONFIG_VERSION ?= latest

build: # @HELP build the Go binaries and run all validations (default)
build:
	go build -o build/_output/onos-config ./cmd/onos-config

test: # @HELP run the unit tests and source code validation producing a golang style report
test: build deps license_check linters
	go test -race github.com/onosproject/onos-config/...


jenkins-test: build-tools # @HELP run the unit tests and source code validation producing a junit style report for Jenkins
jenkins-test: build deps license_check linters
	TEST_PACKAGES=github.com/onosproject/onos-config/... ./../build-tools/build/jenkins/make-unit

coverage: # @HELP generate unit test coverage data
coverage: build deps linters license_check
	./../build-tools/build/coveralls/coveralls-coverage onos-config

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

linters: golang-ci # @HELP examines Go source code and reports coding problems
	golangci-lint run --timeout 5m

build-tools: # @HELP install the ONOS build tools if needed
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi

jenkins-tools: # @HELP installs tooling needed for Jenkins
	cd .. && go get -u github.com/jstemmer/go-junit-report && go get github.com/t-yuki/gocover-cobertura

golang-ci: # @HELP install golang-ci if not present
	golangci-lint --version || curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b `go env GOPATH`/bin v1.42.0

license_check: build-tools # @HELP examine and ensure license headers exist
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR}

helmit-gnmi: # @HELP run helmit tests locally
	(kubectl delete ns test || exit 0) && kubectl create ns test && helmit test -n test ./cmd/onos-config-tests -c ../onos-helm-charts --suite gnmi

gofmt: # @HELP run the Go format validation
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/ cmd/ tests/)"

onos-config-docker: # @HELP build onos-config base Docker image
	docker build . -f build/onos-config/Dockerfile \
		--build-arg ONOS_MAKE_TARGET=build \
		-t ${DOCKER_REPOSITORY}onos-config:${ONOS_CONFIG_VERSION}

config-model-registry-docker: # @HELP build config-model-registry Docker image
	docker build . -f build/config-model-registry/Dockerfile \
		-t onosproject/config-model-registry:${ONOS_CONFIG_VERSION}

images: # @HELP build all Docker images
images: build onos-config-docker config-model-registry-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images kind-only

kind-only: # @HELP deploy the image without rebuilding first
kind-only:
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image --name ${KIND_CLUSTER_NAME} ${DOCKER_REPOSITORY}onos-config:${ONOS_CONFIG_VERSION}

all: build images

publish: # @HELP publish version on github and dockerhub
	./../build-tools/publish-version ${VERSION} onosproject/onos-config

jenkins-publish: build-tools jenkins-tools # @HELP Jenkins calls this to publish artifacts
	./build/bin/push-images
	../build-tools/release-merge-commit
	../build-tools/build/docs/push-docs

bumponosdeps: # @HELP update "onosproject" go dependencies and push patch to git.
	./../build-tools/bump-onos-deps ${VERSION}

clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-config/onos-config ./cmd/onos/onos
	go clean -testcache github.com/onosproject/onos-config/...

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
