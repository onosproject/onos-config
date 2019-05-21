# Running onos-config 

> The commands shown below can be run from anywhere on your PC provided that go tools are installed
> and the GOPATH environment variable is set, e.g. `export GOPATH=~/go`

## Run Server Locally
The onos-config server can be run as follows:
```bash
go run github.com/onosproject/onos-config/cmd/onos-config-manager \
    -configStore=$HOME/go/src/github.com/onosproject/onos-config/configs/configStore-sample.json \
    -changeStore=$HOME/go/src/github.com/onosproject/onos-config/configs/changeStore-sample.json \
    -deviceStore=$HOME/go/src/github.com/onosproject/onos-config/configs/deviceStore-sample.json \
    -networkStore=$HOME/go/src/github.com/onosproject/onos-config/configs/networkStore-sample.json
```

## Run Server in Docker Image
Alternatively, to run onos-config via its Docker image like this:
```
docker run -p 5150:5150 -v `pwd`/configs:/etc/onos-config-manager -it onosproject/onos-config \
    -configStore=/etc/onos-config-manager/configStore-sample.json \
    -changeStore=/etc/onos-config-manager/changeStore-sample.json \
    -deviceStore=/etc/onos-config-manager/deviceStore-sample.json \
    -networkStore=/etc/onos-config-manager/networkStore-sample.json
```
Note that the local config directory is mounted from the container to allow access to local
test configuration files. You can [build your own version of the onos-config Docker image](build.md) 
or use the published one.


## Northbound Get Request via gNMI
__onos-config__ extends standard gNMI as a method of accessing a complete
configuration system consisting of *several* devices - each identified by _target_.
It supports network wide configuration actions (multiple 
updates on multiple devices at once, and rollback of same).

The gNMI Northbound interface is available through https on port 5150.

### gnmi_cli utility
A simple way to issue a gNMI Get request is to use the `gnmi_cli` utility from
the [OpenConfig](https://github.com/openconfig/gnmi) project. If it's not on your system, install as follows:
```bash
go get -u github.com/openconfig/gnmi/cmd/gnmi_cli
```
> For troubleshooting information see [gnmi_user_manual.md](../tools/test/devicesim/gnmi_user_manual.md)

### A simple Get operation
Use `gnmi_cli -get` to get configuration for a particular device (target) from the system.
> Use "target" as the identifier of the device, and the "elem" collection is the path to the requested element.
> If config from several devices are required, several paths can be added
```bash
gnmi_cli -get -address localhost:5150 \
    -proto "path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem:<name:'config'> elem: <name: 'motd-banner'>>" \
    -timeout 5s -alsologtostderr\
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt
```

### List all device names (targets)
A useful way to retrieve all stored device names is with the command:
```bash
gnmi_cli -get -address localhost:5150 \
    -proto "path: <target: '*'>" \
    -timeout 5s -alsologtostderr \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt
```

> The value in the response can be an individual value or a tree of values depending
> on the scope of the request.

### List complete configuration for a device (target)
>Use the following value for proto to get all configuration and operational state on a particular device
>    -proto "path: <target: 'localhost:10161'>"

### Get a keyed index in a list
Use a proto value like:
>    -proto "path: <target: 'localhost:10161',
>         elem: <name: 'openconfig-system:system'>
>         elem: <name: 'openconfig-openflow:openflow'> elem: <name: 'controllers'>
>         elem: <name: 'controller' key: <key: 'name' value: 'main'>>
>         elem: <name: 'connections'> elem: <name: 'connection' key: <key: 'aux-id' value: '0'>>
>         elem: <name: 'config'> elem: <name: 'address'>>"

## Northbound Set Request via gNMI
Similarly, to make a gNMI Set request, use the `gnmi_cli -set` command as in the example below:

```bash
gnmi_cli -address localhost:5150 -set \
    -proto "update: <path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>> val: <string_val: 'Europe/Paris'>>" \
    -timeout 5s -alsologtostderr \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt
```

> The corresponding -get for this will use the -proto
> "path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>"

> Currently (May '19) no checking of the contents is enforced when doing a Set operation
> and the config is forwarded down to the southbound layer only if a device is registered
> in the topocache (currently in the deviceStore)

## Northbound Subscribe Once Request via gNMI
Similarly, to make a gNMI Subscribe Once request, use the `gnmi_cli` command as in the example below, 
please note the `1` as subscription mode to indicate to send the response once:

```bash
gnmi_cli -address localhost:5150 \
    -proto "subscribe:<mode: 1, prefix:<>, subscription:<path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s alsologtostderr \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt
```

> This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.

## Northbound Subscribe Request for Stream Notifications via gNMI
Similarly, to make a gNMI Subscribe request for streaming, use the `gnmi_cli` command as in the example below, 
please note the `0` as subscription mode to indicate streaming:

```bash
gnmi_cli -address localhost:5150 \
    -proto "subscribe:<mode: 0, prefix:<>, subscription:<path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -alsologtostderr \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt
```

> This command will block until there is a change at the requested value that gets
> propagated to the underlying stream. Also as per `gnmi_cli` behaviour the updates get printed twice. 

## Administrative Tools
The project provides a number of administrative tools for remotely accessing the enhanced northbound
functionality.

### List Network Changes
For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
go run github.com/onosproject/onos-config/cmd/admin/net-changes \
    -certPath tools/test/devicesim/certs/client1.crt \
    -keyPath tools/test/devicesim/certs/client1.key
```

### Rollback Network Change
To rollback a network use the rollback admin tool. This will rollback the last network
change unless a specific change is given with the **-changename** parameter
```bash
go run github.com/onosproject/onos-config/cmd/admin/rollback \
    -changename Change-VgUAZI928B644v/2XQ0n24x0SjA= \
    -certPath tools/test/devicesim/certs/client1.crt \
    -keyPath tools/test/devicesim/certs/client1.key
```

## Diagnostic Tools
There are a number of commands that provide internal view into the state the onos-config store.
These tools use a special-purpose gRPC interfaces to obtain the internal meta-data
from the running onos-config process. Please note that these tools are intended purely for
diagnostics and should not be relied upon for programmatic purposes as they are not subject
to any backward compatibility guarantees.

For example, run the following to list all changes submitted through the northbound gNMI 
as they are tracked by the system broken-up into device specific batches:
```bash
go run github.com/onosproject/onos-config/cmd/diags/changes \
    -certPath tools/test/devicesim/certs/client1.crt \
    -keyPath tools/test/devicesim/certs/client1.key
```
> For a specific change use the -changeid argument


To get details from the Configuration store use
```bash
go run github.com/onosproject/onos-config/cmd/diags/configs \
    -certPath tools/test/devicesim/certs/client1.crt \
    -keyPath tools/test/devicesim/certs/client1.key
```
> For the configuration for a specific device use the -devicename argument


To get the aggregate configuration of a device from the store use
```bash
go run github.com/onosproject/onos-config/cmd/diags/devicetree \
    -devicename localhost:10161 \
    -certPath tools/test/devicesim/certs/client1.crt \
    -keyPath tools/test/devicesim/certs/client1.key
```

> Of course, there will be many more such commands available in the near future


## Testing Scripts

You can use testing scripts under [Testing Scripts](../tools/test/scripts) directory to run 
all of the above commands.  