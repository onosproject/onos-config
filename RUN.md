# Bare bones of a Config Mgmt system in Go

## Install
```bash
go get github.com/opennetworkinglab/onos-config/onos-config-manager
```

## Unit test
```bash
go test -v github.com/opennetworkinglab/onos-config/store
```

## Run
```bash
go run github.com/opennetworkinglab/onos-config/onos-config-manager \
-configStore=$HOME/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/stores/configStore-sample.json \
-changeStore=$HOME/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/stores/changeStore-sample.json
```
