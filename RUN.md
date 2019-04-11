# Bare bones of a Config Mgmt system in Go

To run:
```
cd ~/go/src/onos-config

pushd store && \
go build store-api.go && \
go install store-api.go && \
go test && \
popd && \
go build manager/config-manager.go && \
./config-manager
```
