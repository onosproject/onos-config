export CGO_ENABLED=0

.PHONY: build

deps: 
	dep ensure -v

build: deps
	export GOOS=linux
	export GOARCH=amd64
	go build -o build/_output/onos-config-manager ./cmd/onos-config-manager

test: deps
	go test github.com/onosproject/onos-config/pkg/...
	golint github.com/onosproject/onos-config/pkg/...
	go vet github.com/onosproject/onos-config/pkg/...

run: deps
	go run cmd/onos-config-manager/config-manager.go