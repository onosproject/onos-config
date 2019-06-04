export CGO_ENABLED=0

.PHONY: build

ONOS_CONFIG_VERSION := latest
ONOS_BUILD_VERSION := stable

all: image

image: # @HELP build onos-config image
	docker run --rm -it -v `pwd`:/go/src/github.com/onosproject/onos-config onosproject/golang-build:${ONOS_BUILD_VERSION} protos
	docker build . -f build/onos-config/Dockerfile \
	--build-arg ONOS_BUILD_VERSION=${ONOS_BUILD_VERSION} \
	-t onosproject/onos-config:${ONOS_CONFIG_VERSION}

deps: # @HELP ensure that the required dependencies are in place
	dep ensure -v
	bash -c "diff -u <(echo -n) <(git diff Gopkg.lock)"

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
build: protos test
	export GOOS=linux
	export GOARCH=amd64
	go build -o build/_output/onos-config ./cmd/onos-config

test: # @HELP run the unit tests
test: deps lint vet license_check gofmt
	go test github.com/onosproject/onos-config/pkg/...
	go test github.com/onosproject/onos-config/cmd/...

coverage: # @HELP generate unit test coverage data
coverage: test
	./tools/build/coveralls-coverage

run: # @HELP run mainline in cmd/onos-config
run: deps
	go run cmd/onos-config/onos-config.go

run-docker: # @HELP run onos-config docker image
run-docker: image
	docker run -d -p 5150:5150 -v `pwd`/configs:/etc/onos-config \
	--name onos-config onosproject/onos-config \
	-configStore=/etc/onos-config/configStore-sample.json \
	-changeStore=/etc/onos-config/changeStore-sample.json \
	-deviceStore=/etc/onos-config/deviceStore-sample.json \
	-networkStore=/etc/onos-config/networkStore-sample.json

clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output
	rm -rf ./vendor
	rm -rf ./cmd/onos-config/onos-config

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '