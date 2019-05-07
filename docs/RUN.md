# Bare bones of a Config Mgmt system in Go

> The commands can be run from anywhere on your PC - it assumes that go is installed
> and your:
> GOPATH=~/go

## Install
```bash
go get https://github.com/openconfig/gnmi && \
go get github.com/onosproject/onos-config/onos-config-manager
```
> This pulls from master branch.
> For the moment (May 19) you should check the project out from Git and use the
> __firststeps__ branch

## Unit test
```bash
go test -v github.com/onosproject/onos-config/pkg/store
go test -v github.com/onosproject/onos-config/pkg/store/change
go test -v github.com/onosproject/onos-config/pkg/listener
go test -v github.com/onosproject/onos-config/pkg/northbound/restconf
```

## Run
```bash
go run github.com/onosproject/onos-config/cmd/onos-config-manager \
-restconfPort=8080 \
-configStore=$HOME/go/src/github.com/onosproject/onos-config/configs/configStore-sample.json \
-changeStore=$HOME/go/src/github.com/onosproject/onos-config/configs/changeStore-sample.json \
-deviceStore=$HOME/go/src/github.com/onosproject/onos-config/configs/deviceStore-sample.json

```

or locally from ~/go/src/github.com/onosproject/onos-config/onos-config-manager/
```bash
go build && go run config-manager.go
```

### CLI
A rudimentary CLI allows mostly read only access to the configuration at present.

To have a change of timezone pushed all the way down to a device

1) run the devicesim simulator as described [here](tools/test/devicesim/README.md)

2) Then tail the syslog of the local PC to see log messages from onos-config

3) Run onos-config-manager and choose m3 option, and choose which device to send to

> In the syslog you should see SetResponse op:UPDATE

You can verify the Set was successful with
```bash
gnmi_cli -address localhost:10161 \
    -get \
    -proto "path: <elem: <name: 'system'> elem:<name:'clock'> elem:<name:'config'> elem: <name: 'timezone-name'>>" \
    -timeout 5s \
    -client_crt certs/client1.crt \
    -client_key certs/client1.key \
    -ca_crt certs/onfca.crt \
    -alsologtostderr
```

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

and browse at [http://localhost:6060/pkg/github.com/onosproject/onos-config/](http://localhost:6060/pkg/github.com/onosproject/onos-config/)
