# How To Run ONOS-Config system

> The commands can be run from anywhere on your PC - it assumes that go is installed
> and your:
> GOPATH=~/go

Note: this assumes you have followed the the [developer workflow](./DEV_WORKFLOW.md) steps done. 

## Unit test
```bash
go test -v github.com/onosproject/onos-config/cmd/... && \
go test -v github.com/onosproject/onos-config/pkg/...
```

## Run
```bash
go run github.com/onosproject/onos-config/cmd/onos-config-manager \
-configStore=$HOME/go/src/github.com/onosproject/onos-config/configs/configStore-sample.json \
-changeStore=$HOME/go/src/github.com/onosproject/onos-config/configs/changeStore-sample.json \
-deviceStore=$HOME/go/src/github.com/onosproject/onos-config/configs/deviceStore-sample.json \
-networkStore=$HOME/go/src/github.com/onosproject/onos-config/configs/networkStore-sample.json

```

### gNMI Northbound
The system implements a gNMI Northbound interface on port 5150
To access it you can run (from onos-config):

To run the gNMI GET on the NBI the gnmi_cli can also be used as follows:
> Where the "target" is the identifier of the device, 
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

> The value in the response can be an individual value or a tree of values depending
> on the scope of the request.

>Use the following value for proto to get all configuration and operational state on a particular device
>    -proto "path: <target: 'localhost:10161', elem: <name:'*'>>"

>To get a keyed index in a list use a syntax like
>    -proto "path: <target: 'localhost:10161',
>         elem: <name: 'system'>
>         elem: <name: 'openflow'> elem: <name: 'controllers'>
>         elem: <name: 'controller' key: <key: 'name' value: 'main'>>
>         elem: <name: 'connections'> elem: <name: 'connection' key: <key: 'aux-id' value: '0'>>
>         elem: <name: 'config'> elem: <name: 'address'>>"

## Documentation
> Documentation is not yet publicy published

Run locally
```bash
godoc -goroot=$HOME/go
``` 

and browse at [http://localhost:6060/pkg/github.com/onosproject/onos-config/](http://localhost:6060/pkg/github.com/onosproject/onos-config/)
