export CGO_ENABLED=0

.PHONY: build

ONOS_CONFIG_VERSION := "latest"
ONOS_BUILD_VERSION := "latest"

all: image

image: # @HELP build onos-config image
	docker run -it -v `pwd`:/go/src/github.com/onosproject/onos-config onosproject/onos-config-build:${ONOS_BUILD_VERSION} protos build
	docker build . -f build/onos-config/Dockerfile -t onosproject/onos-config:${ONOS_CONFIG_VERSION}

deps: # @HELP ensure that the required dependencies are in place
	dep ensure -v

lint: # @HELP run the linters for Go source code
	golint -set_exit_status github.com/onosproject/onos-config/pkg/...
	golint -set_exit_status github.com/onosproject/onos-config/cmd/...

vet: # @HELP examines Go source code and reports suspicious constructs
	go vet github.com/onosproject/onos-config/pkg/...
	go vet github.com/onosproject/onos-config/cmd/...

license_check: # @HELP examine and ensure license headers exist
	./build/licensing/boilerplate.py -v

gofmt: # @HELP run the go format utility against code in the pkg and cmd directories
	bash -c "diff -u <(echo -n) <(gofmt -d pkg/ cmd/)"

protos: # @HELP compile the protobuf files
	./build/dev-docker/compile-protos.sh

build: # @HELP build the go binary in the cmd/onos-config package
build: test
	export GOOS=linux
	export GOARCH=amd64
	go build -o build/_output/onos-config ./cmd/onos-config

test: # @HELP run the unit tests
test: deps lint vet license_check gofmt
	go test github.com/onosproject/onos-config/pkg/...
	go test github.com/onosproject/onos-config/cmd/...

run: # @HELP run mainline in cmd/onos-config
run: deps
	go run cmd/onos-config/onos-config.go

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '