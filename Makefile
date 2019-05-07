export CGO_ENABLED=0

.PHONY: build

deps: 
	dep ensure -v

lint:
	golint -set_exit_status github.com/onosproject/onos-config/pkg/...

vet:
	go vet github.com/onosproject/onos-config/pkg/...

build: deps
	export GOOS=linux
	export GOARCH=amd64
	go build -o build/_output/onos-config-manager ./cmd/onos-config-manager

test: deps lint vet
	go test github.com/onosproject/onos-config/pkg/...

run: deps
	go run cmd/onos-config-manager/config-manager.go