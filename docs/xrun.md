# How To Run ONOS-Config system

> The commands can be run from anywhere on your PC - it assumes that go is installed
> and your:
> GOPATH=~/go

Note: this assumes you have followed the the [developer workflow](xdev_workflow.md) steps done. 

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

### Diagnostic Tools
There are several commands that provide internal view into the state the onos-config store
meta-data. These tools use a special-purpose gRPC interface to obtain the internal meta-data
from the running onos-config process. Please note that these tools are intended purely for
diagnostics and should not be relied upon for programmatic purposes as they are not subject
to any backward compatibility guarantees.

To list all network changes submitted through the northbound gNMI interface run:
```bash
go run github.com/onosproject/onos-config/cmd/diags/net-changes
```

Similarly, run the following to list all changes submitted through the northbound gNMI but
broken-up into device specific batches:
```bash
go run github.com/onosproject/onos-config/cmd/diags/changes
```


## Documentation
> Documentation is not yet publicy published

Run locally
```bash
godoc -goroot=$HOME/go
``` 

and browse at [http://localhost:6060/pkg/github.com/onosproject/onos-config/](http://localhost:6060/pkg/github.com/onosproject/onos-config/)
