export CGO_ENABLED=0
export GO111MODULE=on

.PHONY: build

ONOS_CONFIG_VERSION := latest
ONOS_CONFIG_DEBUG_VERSION := debug
ONOS_BUILD_VERSION := stable

MODELPLUGINS = build/_output/testdevice.so.1.0.0 build/_output/testdevice.so.2.0.0 build/_output/devicesim.so.1.0.0 build/_output/stratum.so.1.0.0

build: # @HELP build the Go binaries and run all validations (default)
build: $(MODELPLUGINS)
	CGO_ENABLED=1 go build -o build/_output/onos-config ./cmd/onos-config
	CGO_ENABLED=1 go build -gcflags "all=-N -l" -o build/_output/onos-config-debug ./cmd/onos-config

build/_output/testdevice.so.1.0.0: modelplugin/TestDevice-1.0.0/modelmain.go modelplugin/TestDevice-1.0.0/testdevice_1_0_0/generated.go
	-CGO_ENABLED=1 go build -o build/_output/testdevice.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-1.0.0
	-CGO_ENABLED=1 go build -o build/_output/testdevice-debug.so.1.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-1.0.0

build/_output/testdevice.so.2.0.0: modelplugin/TestDevice-2.0.0/modelmain.go modelplugin/TestDevice-2.0.0/testdevice_2_0_0/generated.go
	-CGO_ENABLED=1 go build -o build/_output/testdevice.so.2.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-2.0.0
	-CGO_ENABLED=1 go build -o build/_output/testdevice-debug.so.2.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-2.0.0

build/_output/devicesim.so.1.0.0: modelplugin/Devicesim-1.0.0/modelmain.go modelplugin/Devicesim-1.0.0/devicesim_1_0_0/generated.go
	-CGO_ENABLED=1 go build -o build/_output/devicesim.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/Devicesim-1.0.0
	-CGO_ENABLED=1 go build -o build/_output/devicesim-debug.so.1.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/Devicesim-1.0.0

build/_output/stratum.so.1.0.0: modelplugin/Stratum-1.0.0/modelmain.go modelplugin/Stratum-1.0.0/stratum_1_0_0/generated.go
	-CGO_ENABLED=1 go build -o build/_output/stratum.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/Stratum-1.0.0
	-CGO_ENABLED=1 go build -o build/_output/stratum-debug.so.1.0.0 -gcflags "all=-N -l" -buildmode=plugin -tags=modelplugin ./modelplugin/Stratum-1.0.0

test: # @HELP run the unit tests and source code validation
test: build deps linters license_check
	go test github.com/onosproject/onos-config/pkg/...
	go test github.com/onosproject/onos-config/cmd/...
	go test github.com/onosproject/onos-config/modelplugin/...

coverage: # @HELP generate unit test coverage data
coverage: build deps linters license_check
	./build/bin/coveralls-coverage

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

linters: # @HELP examines Go source code and reports coding problems
	golangci-lint run

license_check: # @HELP examine and ensure license headers exist
	./build/licensing/boilerplate.py -v

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
		-t onosproject/onos-config-base:${ONOS_CONFIG_VERSION}
	@rm -rf vendor

onos-config-docker: onos-config-base-docker # @HELP build onos-config Docker image
	docker build . -f build/onos-config/Dockerfile \
		--build-arg ONOS_CONFIG_BASE_VERSION=${ONOS_CONFIG_VERSION} \
		-t onosproject/onos-config:${ONOS_CONFIG_VERSION}

onos-config-debug-docker: onos-config-base-docker # @HELP build onos-config Docker debug image
	docker build . -f build/onos-config-debug/Dockerfile \
		--build-arg ONOS_CONFIG_BASE_VERSION=${ONOS_CONFIG_VERSION} \
		-t onosproject/onos-config:${ONOS_CONFIG_DEBUG_VERSION}

images: # @HELP build all Docker images
images: build onos-config-docker onos-config-debug-docker

kind: # @HELP build Docker images and add them to the currently configured kind cluster
kind: images
	@if [ `kind get clusters` = '' ]; then echo "no kind cluster found" && exit 1; fi
	kind load docker-image onosproject/onos-config:${ONOS_CONFIG_DEBUG_VERSION}

all: build images

clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-config/onos-config ./cmd/onos/onos

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
