export CGO_ENABLED=0

.PHONY: build

build:
	export GOOS=linux
	export GOARCH=amd64
	dep ensure -v
	go build -o build/_output/onos-config-manager ./cmd/onos-config-manager

run:
	dep ensure -v
	go run cmd/onos-config-manager/config-manager.go