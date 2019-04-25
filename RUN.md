# Bare bones of a Config Mgmt system in Go

> The commands can be run from anywhere on your PC - it assumes that go is installed
> and your:
> GOPATH=~/go

## Install
```bash
go get github.com/opennetworkinglab/onos-config/onos-config-manager
```
> This pulls from master branch.
> For the moment (Apr 19) you should check the project out from Git and use the
> __firststeps__ branch

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

or locally from ~/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/
```bash
go build && go run config-manager.go
```

## Documentation
> Documentation is not yet publicy published

Run locally
```bash
godoc -goroot=$HOME/go
``` 

and browse at [http://localhost:6060/pkg/github.com/opennetworkinglab/onos-config/](http://localhost:6060/pkg/github.com/opennetworkinglab/onos-config/)
