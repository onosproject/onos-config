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
    -networkStore=$HOME/go/src/github.com/onosproject/onos-config/configs/networkStore-sample.json \
    -caPath $HOME/go/src/github.com/onosproject/onos-config/tools/test/devicesim/certs/onfca.crt \
    -keyPath $HOME/go/src/github.com/onosproject/onos-config/tools/test/devicesim/certs/localhost.key \
    -certPath $HOME/go/src/github.com/onosproject/onos-config/tools/test/devicesim/certs/localhost.crt
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
The system implements a gNMI Northbound interface on port 5150
To access it you can run (from onos-config):

To issue a gNMI Get request, you can use the `gnmi_cli -get` command as follows:
> Where the "target" is the identifier of the device, 
> and the "elem" collection is the path to the requested element.
> If config from several devices are required several paths can be added
```bash
gnmi_cli -get -address localhost:5150 \
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
>    -proto "path: <target: 'localhost:10161', elem: \<name:'/*'>>"

>To get a keyed index in a list use a syntax like
>    -proto "path: <target: 'localhost:10161',
>         elem: <name: 'system'>
>         elem: <name: 'openflow'> elem: <name: 'controllers'>
>         elem: <name: 'controller' key: <key: 'name' value: 'main'>>
>         elem: <name: 'connections'> elem: <name: 'connection' key: <key: 'aux-id' value: '0'>>
>         elem: <name: 'config'> elem: <name: 'address'>>"

## Northbound Set Request via gNMI
Similarly, to make a gNMI Set request, use the `gnmi_cli -set` command as in the example below:

```bash
gnmi_cli -address localhost:5150 -set \
    -proto "update: <path: <target: 'localhost:10161', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>> val: <string_val: 'Europe/Dublin'>>" \
    -timeout 5s \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr
```

> The corresponding -get for this will use the -proto
> "path: <target: 'localhost:10161', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>"

> Currently no checking of the contents is enforced and the config is not forwarded down to the 
> southbound layer

## Northbound Subscribe Once Request via gNMI
Similarly, to make a gNMI Subscribe Once request, use the `gnmi_cli` command as in the example below:

```bash
gnmi_cli -address localhost:5150 \
    -proto "subscribe:<mode: 1, subscription:<path: <target: 'localhost:10161', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s \
    -client_crt tools/test/devicesim/certs/client1.crt \
    -client_key tools/test/devicesim/certs/client1.key \
    -ca_crt tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr
```

> Currently no checking of the contents is enforced and the config is not forwarded down to the 
> southbound layer

**Note** This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.

## Administrative Tools
The project provides a number of administrative tools for remotely accessing the enhanced northbound
functionality.

For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
go run github.com/onosproject/onos-config/cmd/admin/net-changes
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
go run github.com/onosproject/onos-config/cmd/diags/changes
```

> Of course, there will be many more such commands available in the near future
