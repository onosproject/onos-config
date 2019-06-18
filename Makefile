export CGO_ENABLED=0

.PHONY: build

ONOS_CONFIG_VERSION := latest
ONOS_CONFIG_DEBUG_VERSION := debug
ONOS_BUILD_VERSION := stable

build: # @HELP build the Go binaries and run all validations (default)
build: test
	export GO111MODULE=on
ifndef DEBUG
	CGO_ENABLED=1 go build -o build/_output/onos-config ./cmd/onos-config
else
	CGO_ENABLED=1 go build -gcflags "all=-N -l" -o build/_output/onos-config ./cmd/onos-config
endif
	go build -o build/_output/onos ./cmd/onos
	CGO_ENABLED=1 go build -o build/_output/testdevice.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-1.0.0
	CGO_ENABLED=1 go build -o build/_output/testdevice.so.2.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/TestDevice-2.0.0
	CGO_ENABLED=1 go build -o build/_output/devicesim.so.1.0.0 -buildmode=plugin -tags=modelplugin ./modelplugin/Devicesim-1.0.0

test: # @HELP run the unit tests and source code validation
test: deps lint vet license_check gofmt
	export GO111MODULE=on
	go test github.com/onosproject/onos-config/pkg/...
	go test github.com/onosproject/onos-config/cmd/...
	go test github.com/onosproject/onos-config/modelplugin/...

coverage: # @HELP generate unit test coverage data
coverage: test
	./build/bin/coveralls-coverage

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

lint: # @HELP run the linters for Go source code
	golint -set_exit_status github.com/onosproject/onos-config/pkg/...
	golint -set_exit_status github.com/onosproject/onos-config/cmd/...

vet: # @HELP examines Go source code and reports suspicious constructs
	go vet github.com/onosproject/onos-config/pkg/...
	go vet github.com/onosproject/onos-config/cmd/...
	go vet github.com/onosproject/onos-config/modelplugin/...

license_check: # @HELP examine and ensure license headers exist
	./build/licensing/boilerplate.py -v

gofmt: # @HELP run the Go format validation
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/ cmd/)"

protos: # @HELP compile the protobuf files (using protoc-go Docker)
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config \
		-w /go/src/github.com/onosproject/onos-config \
		--entrypoint pkg/northbound/proto/compile-protos.sh \
		onosproject/protoc-go:stable

onos-config-docker: # @HELP build onos-config Docker image
	docker build . -f build/onos-config/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		-t onosproject/onos-config:${ONOS_CONFIG_VERSION}

onos-config-debug-docker: # @HELP build onos-config Docker debug image
	docker build . -f build/onos-config-debug/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		-t onosproject/onos-config:${ONOS_CONFIG_DEBUG_VERSION}

onos-cli-docker: # @HELP build onos-cli Docker image
	docker build . -f build/onos-cli/Dockerfile \
		--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
		-t onosproject/onos-cli:${ONOS_CONFIG_VERSION}

images: # @HELP build all Docker images
images: onos-config-docker onos-config-debug-docker onos-cli-docker

all: build images

run-docker: # @HELP run onos-config docker image
run-docker: onos-config-docker
	docker stop onos-config || echo "onos-config was not running"
	docker run -d --rm -p 5150:5150 -v `pwd`/configs:/etc/onos-config \
		--name onos-config onosproject/onos-config \
		-configStore=/etc/onos-config/configStore-sample.json \
		-changeStore=/etc/onos-config/changeStore-sample.json \
		-deviceStore=/etc/onos-config/deviceStore-sample.json \
		-networkStore=/etc/onos-config/networkStore-sample.json \
		-modelPlugin=/usr/local/lib/testdevice.so.1.0.0 \
		-modelPlugin=/usr/local/lib/testdevice.so.2.0.0 \
		-modelPlugin=/usr/local/lib/devicesim.so.1.0.0

clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor ./cmd/onos-config/onos-config ./cmd/onos/onos
	docker stop onos-config || echo "onos-config was not running"

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '
