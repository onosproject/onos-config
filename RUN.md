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
go test -v github.com/opennetworkinglab/onos-config/store/change
go test -v github.com/opennetworkinglab/onos-config/listener
go test -v github.com/opennetworkinglab/onos-config/northbound/restconf
```

## Run
```bash
go run github.com/opennetworkinglab/onos-config/onos-config-manager \
-restconfPort=8080 \
-configStore=$HOME/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/stores/configStore-sample.json \
-changeStore=$HOME/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/stores/changeStore-sample.json \
-deviceStore=$HOME/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/stores/deviceStore-sample.json

```

or locally from ~/go/src/github.com/opennetworkinglab/onos-config/onos-config-manager/
```bash
go build && go run config-manager.go
```

### CLI
A rudimentary CLI allows mostly read only access to the configuration at present.

### Restconf
A rudimentary read-only Restconf interface is given at 
* http://localhost:8080/restconf/

To list changes:
http://localhost:8080/restconf/change/

To list configurations:
* http://localhost:8080/restconf/configuration/

To list data in tree format:
* http://localhost:8080/restconf/data/

To listen to an event stream in Server Sent Events format:
* http://localhost:8080/restconf/events/
> curl -v -H "Accept: text/event-stream" -H "Connection: keep-alive" -H "Cache-Control: no-cache" http://localhost:8080/restconf/events/

> An event arrives when a change is made (e.g. at the CLI m1 or m2)



## Documentation
> Documentation is not yet publicy published

Run locally
```bash
godoc -goroot=$HOME/go
``` 

and browse at [http://localhost:6060/pkg/github.com/opennetworkinglab/onos-config/](http://localhost:6060/pkg/github.com/opennetworkinglab/onos-config/)
