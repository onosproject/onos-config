# Config Mgmt system in Go

> The commands can be run from anywhere on your PC - it assumes that go is installed
> and your:
> GOPATH=~/go

## Install
See [build/dev-docker/README.md](/../master/build/dev-docker/README.md) for instructions
to build a Dev image that downloads any dependencies to you local folder
This resolves any go dependencies

## Unit test
```bash
go test -v github.com/onosproject/onos-config/...
```

## Run
```bash
go run github.com/onosproject/onos-config/cmd/onos-config-manager \
-configStore=$HOME/go/src/github.com/onosproject/onos-config/configs/configStore-sample.json \
-changeStore=$HOME/go/src/github.com/onosproject/onos-config/configs/changeStore-sample.json \
-deviceStore=$HOME/go/src/github.com/onosproject/onos-config/configs/deviceStore-sample.json \
-networkStore=$HOME/go/src/github.com/onosproject/onos-config/configs/networkStore-sample.json

```


### CLI
A rudimentary CLI allows mostly read only access to the configuration at present.

To have a change of timezone pushed all the way down to a device

1) run the devicesim simulator as described [here](/../master/tools/test/devicesim/README.md)

2) Then tail the syslog of the local PC to see log messages from onos-config

3) Run onos-config-manager and choose m3 option, and choose which device to send to

> In the syslog you should see SetResponse op:UPDATE

You can verify the Set was successful with (from onos-config)
```bash
gnmi_cli -address localhost:10161 \
    -get \
    -proto "path: <elem: <name: 'system'> elem:<name:'clock'> elem:<name:'config'> elem: <name: 'timezone-name'>>" \
    -timeout 5s \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr
```

### gNMI Northbound
The system implements a gNMI Northbound interface on port 5150
To access it you can run (from onos-config):
```bash
gnmi_cli -address localhost:5150 -capabilities \
    -timeout 5s \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr
```

To run the gNMI GET on the NBI the gnmi_cli can also be used as follows:
> Where the "target" in "path" is the identifier of the device, 
> and the "elem" collection is the path to the requested element.
> If config from several devices are required several paths can be added
```bash
gnmi_cli -get -address localhost:5150 \
    -insecure  \
    -proto "path: <target: 'localhost:10161', elem: <name: 'system'> elem:<name:'config'> elem: <name: 'motd-banner'>>" \
    -timeout 5s \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr
```

> The value in the response can be an individual value (per path)
> or a tree of values per path.

## Documentation
> Documentation is not yet publicy published

Run locally
```bash
godoc -goroot=$HOME/go
``` 

and browse at [http://localhost:6060/pkg/github.com/onosproject/onos-config/](http://localhost:6060/pkg/github.com/onosproject/onos-config/)
