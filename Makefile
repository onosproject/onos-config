export CGO_ENABLED=0

.PHONY: build deploy

build:
	export GOOS=linux
	export GOARCH=amd64
	dep ensure -v
	go build -o build/_output/onos-config-manager ./onos-config-manager

run:
	dep ensure -v
	go run onos-config-manager/config-manager.go