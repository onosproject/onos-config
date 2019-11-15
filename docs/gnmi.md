# Northbound gNMI service
The system provides a Northbound gNMI service.

gNMI extensions supported on the Northbound are described in [gnmi_extensions.md](./gnmi_extensions.md) 

## gnmi_cli utility
A simple way to issue a gNMI requests is to use the `gnmi_cli` utility from
the [OpenConfig](https://github.com/openconfig/gnmi) project.

### gnmi_cli utility through 'onit'
> On a deployed cluster the onos-cli pod has a gNMI client that can be used to
> format and send gNMI messages.
To access the CLI use
```
onit onos-cli
```
to get in to the `onos-cli` pod and then run gnmi_cli from there.

### Accessing from local machine
An alternative is to install on your system, install as follows:
```bash
go get -u github.com/openconfig/gnmi/cmd/gnmi_cli
```
> For troubleshooting information see [gnmi_user_manual.md](https://github.com/onosproject/simulators/blob/master/docs/gnmi/gnmi_user_manual.md)
## Namespaces
__onos-config__ follows the YGOT project in simplification by not using namespaces in paths. This can be achieved 
because the YANG models used do not have clashing device names that need to be qualified by namespaces. 
This helps developers, avoiding un-needed complication and redundancy. 

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
gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

### List all device names (targets)
A useful way to retrieve all stored device names is with the command:
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: '*'>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

> The value in the response can be an individual value or a tree of values depending
> on the scope of the request.

### List complete configuration for a device (target)
>Use the following value for proto to get all configuration and operational state on a particular device
>    -proto "path: <target: 'devicesim-1'>"

### Get a keyed index in a list
Use a proto value like:
>    -proto "path: <target: 'devicesim-1',
>         elem: <name: 'system'>
>         elem: <name: 'openflow'> elem: <name: 'controllers'>
>         elem: <name: 'controller' key: <key: 'name' value: 'main'>>
>         elem: <name: 'connections'> elem: <name: 'connection' key: <key: 'aux-id' value: '0'>>
>         elem: <name: 'config'> elem: <name: 'address'>>"

### Use wildcards in a path
onos-config supports the wildcards `*` and `...` in gNMI paths, meaning match
one item of match all items respectively as defined in the gNMI
[specification](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md#wildcards-in-paths).

For instance to retrieve all instances of an interface use `*` as the key: 
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path:<target: 'devicesim-1', elem:<name:'interfaces' > elem:<name:'interface' key:<key:'name' value:'*' > > elem:<name:'config'> elem:<name:'enabled' >>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
> This returns the `enabled` config attribute of both interfaces 'eth1' and 'admin'

To retrieve both the config and state values of both then additionally the use
`*` in place of `config`:
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path:<target: 'devicesim-1', elem:<name:'interfaces' > elem:<name:'interface' key:<key:'name' value:'*' > > elem:<name:'*'> elem:<name:'enabled' >>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
> If the device is connected and the OperationState cache is populated this returns
> 4 values - `eth1` config and state enabled values and `admin` config and
> state enabled values.

### Device read only state get
To retrieve state, non-configurable values, there is no difference with a normal gNMI get request.
An example follows:
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: 'devicesim-1',elem:<name:'system' > elem:<name:'openflow' > elem:<name:'controllers' > elem:<name:'controller' key:<key:'name' value:'main' > > elem:<name:'connections' > elem:<name:'connection' key:<key:'aux-id' value:'0' > > elem:<name:'state' > elem:<name:'address'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
## Northbound Set Request via gNMI
Similarly, to make a gNMI Set request, use the `gnmi_cli -set` command as in the example below:

```bash
gnmi_cli -address onos-config:5150 -set \
    -proto "update: <path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>> val: <string_val: 'Europe/Paris'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
giving a response like
```bash
response: <
  path: <
    elem: <
      name: "system"
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
    target: "devicesim-1"
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
>
> If a specific name is desired for a Network Change, the set may be given in the
SetRequest() with the 100 extension at the end of the -proto section like:
> `, extension: <registered_ext: <id: 100, msg: 'myfirstchange'>>`
> See [gnmi_extensions.md](./gnmi_extensions.md) for more on gNMI extensions supported.
>
> The corresponding -get for this require using the -proto
> `path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>`
>
> Currently (Nov '19) checking of the contents done only when a Model Plugin is
> loaded for the device type. 2 checks are done
>
>   1. that a attempt is not being made to change a readonly attribute and
>   2. that valid data types and values are being used.
>
> The config is only forwarded down to the southbound layer only if the config is
> correct and the device is registered in the topocache (currently in the deviceStore)

### Target device not known/creating a new device target
If the `target` device is not currently known to `onos-config` the system will store the configuration internally and apply
it to the `target` device when/if it becomes available.

When the `target` becomes available `onos-config` will compute the latest configuration for it based on the set of 
applied changes and push it to the `target` with a standard `set` operation.

In the case where the `target` device is not known, a special feature of onos-config
has to be invoked to tell the system the type and version to use as a model plugin
for validation - these are given in extensions [101](./gnmi_extensions.md) (version)
and [102](./gnmi_extensions.md) (type).
> This can be used to pre-provision new devices or new versions of devices before
> they are available in the `onos-topo` topology.  

For example using the
gnmi_cli:
```bash
gnmi_cli -address onos-config:5150 -set \
    -proto "update: <path: <target: 'new-device', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>> val: <string_val: 'Europe/Paris'>>, extension: <registered_ext: <id: 100, msg: 'my2ndchange'>>  , extension <registered_ext: <id: 101, msg: '1.0.0'>>, extension: <registered_ext: <id: 102, msg: 'Devicesim'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

> There are restrictions on the use of these extensions in this context:
> * All targets specified in this `set` command will have to be of the same type
> and version as given in extension 101 and 102, even if they already exist on
> the system.

## Northbound Delete Request via gNMI
A delete request in gNMI is done using the set request with `delete` paths instead of `update` or `replace`.
To make a gNMI Set request do delete a path, use the `gnmi_cli -set` command as in the example below:

```bash
gnmi_cli -address onos-config:5150 -set \
    -proto "delete: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

## Northbound Subscribe Request for Stream Notifications via gNMI
Similarly, to make a gNMI Subscribe request for streaming, use the `gnmi_cli` command as in the example below, 
please note the `0` as subscription mode to indicate streaming:

```bash
gnmi_cli -address onos-config:5150 \
    -proto "subscribe:<mode: 0, prefix:<>, subscription:<path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

> This command will block until there is a change at the requested value that gets
> propagated to the underlying stream. Also as per `gnmi_cli` behaviour the updates get printed twice. 

## Northbound Subscribe Once Request via gNMI
Similarly, to make a gNMI Subscribe Once request, use the `gnmi_cli` command as in the example below, 
please note the `1` as subscription mode to indicate to send the response once:

```bash
gnmi_cli -address onos-config:5150 \
    -proto "subscribe:<mode: 1, prefix:<>, subscription:<path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

> This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.

## Northbound Subscribe Poll Request via gNMI
Similarly, to make a gNMI Subscribe POLL request, use the `gnmi_cli` command as in the example below, 
please note the `2` as subscription mode to indicate to send the response in a polling way every `polling_interval` specified seconds:

```bash
gnmi_cli -address onos-config:5150 -polling_interval 5s \
    -proto "subscribe:<mode: 2, prefix:<>, subscription:<sample_interval: 5, path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
> This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.
