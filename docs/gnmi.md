# Northbound gNMI service
The system provides a Northboudn gNMI service. 

## gnmi_cli utility
A simple way to issue a gNMI requests is to use the `gnmi_cli` utility from
the [OpenConfig](https://github.com/openconfig/gnmi) project. If it's not on your system, install as follows:
```bash
go get -u github.com/openconfig/gnmi/cmd/gnmi_cli
```
> For troubleshooting information see [gnmi_user_manual.md](https://github.com/onosproject/simulators/blob/master/docs/gnmi/gnmi_user_manual.md)

## Northbound gNMI Get Request 
__onos-config__ extends standard gNMI as a method of accessing a complete
configuration system consisting of *several* devices - each identified by _target_.
It supports network wide configuration actions (multiple 
updates on multiple devices at once, and rollback of same).

The gNMI Northbound interface is available through https on port 5150.

### A simple Get operation
Use `gnmi_cli -get` to get configuration for a particular device (target) from the system.
> Use "target" as the identifier of the device, and the "elem" collection is the path to the requested element.
> If config from several devices are required, several paths can be added
```bash
gnmi_cli -get -address localhost:5150 \
    -proto "path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem:<name:'config'> elem: <name: 'motd-banner'>>" \
    -timeout 5s -alsologtostderr\
    -client_crt pkg/southbound/testdata/client1.crt \
    -client_key pkg/southbound/testdata/client1.key \
    -ca_crt pkg/southbound/testdata/onfca.crt
```

### List all device names (targets)
A useful way to retrieve all stored device names is with the command:
```bash
gnmi_cli -get -address localhost:5150 \
    -proto "path: <target: '*'>" \
    -timeout 5s -alsologtostderr \
    -client_crt pkg/southbound/testdata/client1.crt \
    -client_key pkg/southbound/testdata/client1.key \
    -ca_crt pkg/southbound/testdata/onfca.crt
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
    -client_crt pkg/southbound/testdata/client1.crt \
    -client_key pkg/southbound/testdata/client1.key \
    -ca_crt pkg/southbound/testdata/onfca.crt
```
giving a response like
```bash
response: <
  path: <
    elem: <
      name: "openconfig-system:system"
    >
    elem: <
      name: "clock"
    >
    elem: <
      name: "config"
    >
    elem: <
      name: "timezone-name"
    >
    target: "localhost:10161"
  >
  op: UPDATE
>
timestamp: 1559122191
extension: <
  registered_ext: <
    id: 100
    msg: "happy_matsumoto"
  >
>
```

> The result will include a field as a gNMI SetResponse extension 100
> giving randomly generated Network Change identifier, which may be subsequently used
> to rollback the change.

> If a specific name is desired for a Network Change, the set may be given in the
SetRequest() with the 100 extension at the end of the -proto section like:
> ", extension: <registered_ext: <id: 100, msg: 'myfirstchange'>>"

> The corresponding -get for this require using the -proto
> "path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>"

> Currently (May '19) no checking of the contents is enforced when doing a Set operation
> and the config is forwarded down to the southbound layer only if a device is registered
> in the topocache (currently in the deviceStore)

## Northbound Subscribe Request for Stream Notifications via gNMI
Similarly, to make a gNMI Subscribe request for streaming, use the `gnmi_cli` command as in the example below, 
please note the `0` as subscription mode to indicate streaming:

```bash
gnmi_cli -address localhost:5150 \
    -proto "subscribe:<mode: 0, prefix:<>, subscription:<path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -alsologtostderr \
    -client_crt pkg/southbound/testdata/client1.crt \
    -client_key pkg/southbound/testdata/client1.key \
    -ca_crt pkg/southbound/testdata/onfca.crt
```

> This command will block until there is a change at the requested value that gets
> propagated to the underlying stream. Also as per `gnmi_cli` behaviour the updates get printed twice. 

## Northbound Subscribe Once Request via gNMI
Similarly, to make a gNMI Subscribe Once request, use the `gnmi_cli` command as in the example below, 
please note the `1` as subscription mode to indicate to send the response once:

```bash
gnmi_cli -address localhost:5150 \
    -proto "subscribe:<mode: 1, prefix:<>, subscription:<path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -alsologtostderr \
    -client_crt pkg/southbound/testdata/client1.crt \
    -client_key pkg/southbound/testdata/client1.key \
    -ca_crt pkg/southbound/testdata/onfca.crt
```

> This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.

## Northbound Subscribe Poll Request via gNMI
Similarly, to make a gNMI Subscribe POLL request, use the `gnmi_cli` command as in the example below, 
please note the `2` as subscription mode to indicate to send the response in a polling way every `polling_interval` specified seconds:

```bash
gnmi_cli -address localhost:5150 \
     -proto "subscribe:<mode: 2, prefix:<>, subscription:<sample_interval: 5, path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
     -timeout 5s \
     -polling_interval 5s \
     -client_crt pkg/southbound/testdata/client1.crt \
     -client_key pkg/southbound/testdata/client1.key \
     -ca_crt pkg/southbound/testdata/onfca.crt
```
> This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.