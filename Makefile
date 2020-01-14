export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_VERSION := latest
ONOS_CONFIG_DEBUG_VERSION := debug
ONOS_BUILD_VERSION := stable

MODELPLUGINS = build/_output/testdevice.so.1.0.0 build/_output/testdevice.so.2.0.0 build/_output/devicesim.so.1.0.0 build/_output/stratum.so.1.0.0
MODELPLUGINSDEBUG = build/_output/testdevice-debug.so.1.0.0 build/_output/testdevice-debug.so.2.0.0 build/_output/devicesim-debug.so.1.0.0 build/_output/stratum-debug.so.1.0.0

build: # @HELP build the Go binaries and run all validations (default)
build:
	CGO_ENABLED=1 go build -o build/_output/onos-config ./cmd/onos-config

build-debug: # @HELP build the Go binaries and run all validations (default)
build-debug:
	CGO_ENABLED=1 go build -gcflags "all=-N -l" -o build/_output/onos-config-debug ./cmd/onos-config

build-plugins: # @HELP build plugin binaries
build-plugins: $(MODELPLUGINS)

build-plugins-debug: # @HELP build plugin binaries
build-plugins-debug: $(MODELPLUGINSDEBUG)

build/_output/testdevice.so.1.0.0: modelplugin/TestDevice-1.0.0/modelmain.go modelplugin/TestDevice-1.0.0/testdevice_1_0_0/generated.go
	CGO_ENABLED=1 go build -o build/_output/testdevice.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-1.0.0

build/_output/testdevice-debug.so.1.0.0: modelplugin/TestDevice-1.0.0/modelmain.go modelplugin/TestDevice-1.0.0/testdevice_1_0_0/generated.go
	CGO_ENABLED=1 go build -o build/_output/debug/testdevice-debug.so.1.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-1.0.0

build/_output/testdevice.so.2.0.0: modelplugin/TestDevice-2.0.0/modelmain.go modelplugin/TestDevice-2.0.0/testdevice_2_0_0/generated.go
	CGO_ENABLED=1 go build -o build/_output/testdevice.so.2.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-2.0.0

build/_output/testdevice-debug.so.2.0.0: modelplugin/TestDevice-2.0.0/modelmain.go modelplugin/TestDevice-2.0.0/testdevice_2_0_0/generated.go
    CGO_ENABLED=1 go build -o build/_output/debug/testdevice-debug.so.2.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-2.0.0

build/_output/devicesim.so.1.0.0: modelplugin/Devicesim-1.0.0/modelmain.go modelplugin/Devicesim-1.0.0/devicesim_1_0_0/generated.go
	CGO_ENABLED=1 go build -o build/_output/devicesim.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/Devicesim-1.0.0

build/_output/devicesim-debug.so.1.0.0: modelplugin/Devicesim-1.0.0/modelmain.go modelplugin/Devicesim-1.0.0/devicesim_1_0_0/generated.go
	CGO_ENABLED=1 go build -o build/_output/debug/devicesim-debug.so.1.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/Devicesim-1.0.0

build/_output/stratum.so.1.0.0: modelplugin/Stratum-1.0.0/modelmain.go modelplugin/Stratum-1.0.0/stratum_1_0_0/generated.go
	CGO_ENABLED=1 go build -o build/_output/stratum.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/Stratum-1.0.0

build/_output/stratum-debug.so.1.0.0: modelplugin/Stratum-1.0.0/modelmain.go modelplugin/Stratum-1.0.0/stratum_1_0_0/generated.go
	CGO_ENABLED=1 go build -o build/_output/debug/stratum-debug.so.1.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/Stratum-1.0.0

test: # @HELP run the unit tests and source code validation
test: build deps linters license_check
	CGO_ENABLED=1 go test -race github.com/onosproject/onos-config/pkg/...
	CGO_ENABLED=1 go test -race github.com/onosproject/onos-config/cmd/...
	CGO_ENABLED=1 go test -race github.com/onosproject/onos-config/modelplugin/...
	CGO_ENABLED=1 go test -race github.com/onosproject/onos-config/api/...

coverage: # @HELP generate unit test coverage data
coverage: build deps linters license_check
	./build/bin/coveralls-coverage

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run --timeout 30m

license_check: # @HELP examine and ensure license headers exist
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR}

gofmt: # @HELP run the Go format validation
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/ cmd/ tests/)"

protos: # @HELP compile the protobuf files (using protoc-go Docker)
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config \
		-w /go/src/github.com/onosproject/onos-config \
		--entrypoint build/bin/compile-protos.sh \
		onosproject/protoc-go:stable

onos-config-base-docker: # @HELP build onos-config base Docker image
	@go mod vendor
	docker build . -f build/base/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		--build-arg ONOS_MAKE_TARGET=build \
		-t onosproject/onos-config-base:${ONOS_CONFIG_VERSION}
	@rm -rf vendor

onos-config-base-debug-docker: # @HELP build onos-config base Docker image
	@go mod vendor
	docker build . -f build/base/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		--build-arg ONOS_MAKE_TARGET=build-debug \
		-t onosproject/onos-config-base:${ONOS_CONFIG_DEBUG_VERSION}
	@rm -rf vendor

onos-config-plugins-docker: # @HELP build onos-config plugins Docker image
	@go mod vendor
	docker build . -f build/plugins/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		--build-arg ONOS_MAKE_TARGET=build-plugins \
		-t onosproject/onos-config-plugins:${ONOS_CONFIG_VERSION}
	@rm -rf vendor

onos-config-plugins-debug-docker: # @HELP build onos-config plugins Docker image
	@go mod vendor
	docker build . -f build/plugins/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		--build-arg ONOS_MAKE_TARGET=build-plugins-debug \
		-t onosproject/onos-config-plugins:${ONOS_CONFIG_DEBUG_VERSION}
	@rm -rf vendor

onos-config-docker: onos-config-base-docker onos-config-plugins-docker # @HELP build onos-config Docker image
	docker build . -f build/onos-config/Dockerfile \
		--build-arg ONOS_CONFIG_BASE_VERSION=${ONOS_CONFIG_VERSION} \
		-t onosproject/onos-config:${ONOS_CONFIG_VERSION}

onos-config-debug-docker: onos-config-base-debug-docker onos-config-plugins-debug-docker # @HELP build onos-config Docker debug image
	docker build . -f build/onos-config-debug/Dockerfile \
		--build-arg ONOS_CONFIG_BASE_VERSION=${ONOS_CONFIG_DEBUG_VERSION} \
		-t onosproject/onos-config:${ONOS_CONFIG_DEBUG_VERSION}

onos-config-tests-docker: # @HELP build onos-config tests Docker image
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/onos-config-tests/_output/bin/onos-config-tests ./cmd/onos-config-tests
	docker build . -f build/onos-config-tests/Dockerfile -t onosproject/onos-config-tests:${ONOS_CONFIG_VERSION}

onos-config-benchmarks-docker: # @HELP build onos-config benchmarks Docker image
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/onos-config-benchmarks/_output/bin/onos-config-benchmarks ./cmd/onos-config-benchmarks
	docker build . -f build/onos-config-benchmarks/Dockerfile -t onosproject/onos-config-benchmarks:${ONOS_CONFIG_VERSION}

images: # @HELP build all Docker images
images: build onos-config-docker onos-config-tests-docker onos-config-benchmarks-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ "`kind get clusters`" = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/onos-config:${ONOS_CONFIG_VERSION}
	kind load docker-image onosproject/onos-config-tests:${ONOS_CONFIG_VERSION}
	kind load docker-image onosproject/onos-config-benchmarks:${ONOS_CONFIG_VERSION}

all: build images

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
