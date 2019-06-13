# Running onos-config 

> The commands shown below can be run from anywhere on your PC provided that go tools are installed
> and the GOPATH environment variable is set, e.g. `export GOPATH=~/go`

## Run Server Locally
The onos-config server can be run as follows:
```bash
go run github.com/onosproject/onos-config/cmd/onos-config \
    -configStore=$HOME/go/src/github.com/onosproject/onos-config/configs/configStore-sample.json \
    -changeStore=$HOME/go/src/github.com/onosproject/onos-config/configs/changeStore-sample.json \
    -deviceStore=$HOME/go/src/github.com/onosproject/onos-config/configs/deviceStore-sample.json \
    -networkStore=$HOME/go/src/github.com/onosproject/onos-config/configs/networkStore-sample.json \
    -modelPlugin=$HOME/go/src/github.com/onosproject/onos-config/modelplugin/TestDevice-1.0.0/testdevice.so.1.0.0 \
    -modelPlugin=$HOME/go/src/github.com/onosproject/onos-config/modelplugin/TestDevice-2.0.0/testdevice.so.1.0.0 \
    -modelPlugin=$HOME/go/src/github.com/onosproject/onos-config/modelplugin/Devicesim-1.0.0/devicesim.so.1.0.0
```

## Run Server in Docker Image
Alternatively, to run onos-config via its Docker image like this:
```
make run-docker
```
Note that the local config directory is mounted from the container to allow access to local
test configuration files. This command will build a docker image from source.

## Northbound gNMI service
The system provides a full implementation of the gNMI spec as a northbound service.

Here is an example on how to use `gnmi_cli -get` to get configuration for a particular device (target) from the system.
```bash
gnmi_cli -get -address localhost:5150 \
    -proto "path: <target: 'localhost-1', elem: <name: 'openconfig-system:system'> elem:<name:'config'> elem: <name: 'motd-banner'>>" \
    -timeout 5s -alsologtostderr\
    -client_crt pkg/southbound/testdata/client1.crt \
    -client_key pkg/southbound/testdata/client1.key \
    -ca_crt pkg/southbound/testdata/onfca.crt
```
[Full list of the gNMI northbound endpoints](gnmi.md)

## Administrative and Diagnostic Tools
The project provides enhanced northbound functionality though administrative and 
diagnostic tools, which are integrated into the consolidated `onos` command.

For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
go run github.com/onosproject/onos-config/cmd/onos net-changes
```

Or, run the following to list all changes submitted through the northbound gNMI 
as they are tracked by the system broken-up into device specific batches:
```bash
go run github.com/onosproject/onos-config/cmd/onos changes
```

You can read more comprehensive documentation of the various 
[administrative and diagnostic commands](cli.md).
