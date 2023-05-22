<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# Northbound gNMI service
The system provides a Northbound gNMI service. gNMI is a specialization of gRPC,
specifically for configuration of systems or devices. In `onos-config` the gNMI
interface is secured through TLS, and is made available on port **5150**.

gNMI extensions supported on the Northbound are described in [gnmi_extensions.md](./gnmi_extensions.md)

## gnmi_cli utility
A simple way to issue a gNMI requests is to use the `gnmi_cli` utility from
the [OpenConfig](https://github.com/openconfig/gnmi) project.

> A special version of this tool that can connect over a plain connection is available
> with `go get github.com/onosproject/onos-cli/cmd/gnmi_cli`. This version gives
> the extra `-encodingType` (`-en`) and `-tlsDisabled` (`-tls`) options.

More instructions including all the examples below can be found in
[gnmi_cli tool examples](https://github.com/onosproject/onos-config/tree/master/gnmi_cli).

### gnmi_cli utility through onos-cli
> On a deployed cluster the onos-cli pod has this gNMI client installed.

You can run the following command to get in to the **onos-cli** pod and then run gnmi_cli from there:

```bash
kubectl -n micro-onos exec -it deployment/onos-cli -- /bin/bash
```

### Accessing from local machine
An alternative is to install on your system, install as follows:
```bash
go get -u github.com/onosproject/onos-cli/cmd/gnmi_cli
```

Then you can use k8s port forwarding to run gnmi_cli locally on your machine as follows:

```bash
kubectl port-forward -n <onos-namespace> <onos-config-pod-id> 5150:5150
```

> For troubleshooting information see [gnmi_user_manual.md](https://github.com/onosproject/gnxi-simulators/blob/master/docs/gnmi/gnmi_user_manual.md)

## Namespaces
__onos-config__ follows the YGOT project in simplification by not using namespaces in paths. This can be achieved
because the YANG models used do not have clashing device names that need to be qualified by namespaces.
This helps developers, avoiding un-needed complication and redundancy.

## Capabilities
For example use `gnmi_cli -capabilities` to get the capabilities from the system.

```bash
gnmi_cli -capabilities --address=onos-config:5150 \
  -timeout 5s -insecure \
  -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

The Encodings supported are `JSON`, `JSON_IETF`, and `PROTO`.

> This returns the aggregate of all of the model plugins and their versions
> that have been loaded.
>
> Here the certificate locations are inside the `onos-cli` pod.
> If the CA does not exactly match the cert inside `onos-config` and the hostname
> of the server does not match the cert it is necessary to use the `-insecure`
> flag. Encryption is still used in this case.

## Northbound Set Request via gNMI
To make a gNMI Set request, use the `gnmi_cli -set` command as in the example below:

> Since the onos-config data store is empty by default, the Set examples are shown
> before the Get examples (below).
> 
> By default `onos-config` does not have any targets - it gets these from `onos-topo`. 
> See [onos-topo](https://docs.onosproject.org/onos-topo/docs/cli/)
> for how to create a `devicesim-1` target on the system, prior to the `Set` below.

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/set.timezone.gnmi)
```bash
gnmi_cli -address onos-config:5150 -set \
    -proto "update: <path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>> val: <string_val: 'Europe/Paris'>>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
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
timestamp: 1684667662
extension: <
  registered_ext: <
    id: 110
    msg: "\n)uuid:5da2396a-10c3-40d6-a841-007beabd1f4f\020\003"
  >
>
```

> The result will include a field as a gNMI SetResponse extension 110
> giving randomly generated Network Change identifier.

### Open Config models e.g. devicesim 1.0.x

Adding an "eth2" to each of the device

> There is a quirk with the OpenConfig models (e.g. for Stratum and Devicesim),
> where the name of the interface is a [leaf ref](https://github.com/openconfig/public/blob/e3c0374ce6aa9d1230ea31a5f0f9a739ed0db308/release/models/interfaces/openconfig-interfaces.yang#L164)
> to a name attribute beneath it. This means that an interface cannot be created
> without specifying the config/name attribute at the same time (as above). Otherwise
> error `rpc error: code = InvalidArgument desc = pointed-to value with path ../config/name from field Name value eth2 (string ptr) schema /device/interfaces/interface/name is empty set`
> will occur.

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/set.eth2.gnmi)
```bash
gnmi_cli -address onos-config:5150 -set \
    -proto "update: <path: <target: 'devicesim-1', elem: <name: 'interfaces'> elem: <name: 'interface' key:<key:'name' value:'eth2' >> elem: <name: 'config'> elem: <name: 'name'>> val: <string_val: 'eth2'>>" \
    -timeout 5s -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt \
    -client_key /etc/ssl/certs/client1.key \
    -ca_crt /etc/ssl/certs/onfca.crt
```

## Northbound gNMI Get Request
__onos-config__ extends standard gNMI as a method of accessing a complete
configuration system consisting of *several* devices - each identified by _target_.

The gNMI Northbound interface is available through https on port 5150.

> As described in [Key Concepts](run.md), even if the `device-simulator` is connected
> the configuration in `onos-config` will be empty as no initial synchronization
> is done. A Set operation is necessary before a Get will show any results.

### A simple Get operation
Use `gnmi_cli -get` to get configuration for a particular device (target) from the system.
> Use "target" as the identifier of the device, and the "elem" collection is the path to the requested element.
> If config from several devices are required, several paths can be added

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/get.timezone.gnmi)
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

### List all device names (targets)
A useful way to retrieve all stored device names is with the command:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/get.alldevices.gnmi)
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: '*'>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

Here the encoding requested was `PROTO` which will return the values in a Lef List.
Alternatively `JSON` could have been used, which will give a JSON payload in a JSON_Val.

### List complete configuration for a device (target)

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/get.devicesim1.gnmi)
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: 'devicesim-1'>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
> Here all `elem` components are omitted, which is like requesting '/'.

### Get a keyed index in a list
Use a proto value like:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/get.connections.gnmi)
```
-proto "path: <target: 'devicesim-1',
         elem: <name: 'system'>
         elem: <name: 'openflow'> elem: <name: 'controllers'>
         elem: <name: 'controller' key: <key: 'name' value: 'main'>>
         elem: <name: 'connections'> elem: <name: 'connection' key: <key: 'aux-id' value: '0'>>
         elem: <name: 'config'> elem: <name: 'address'>>"
```

### Use wildcards in a path
`onos-config` supports the wildcards `*` and `...` in gNMI paths, meaning match
one item of match all items respectively as defined in the gNMI
[specification](https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md#wildcards-in-paths).

For instance to retrieve all instances of an interface use `*` as the key:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/get.interfaceswc.gnmi)
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path:<target: 'devicesim-1', elem:<name:'interfaces' > elem:<name:'interface' key:<key:'name' value:'*' > > elem:<name:'config'> elem:<name:'enabled' >>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
> This returns the `enabled` config attribute of both interfaces 'eth1' and 'admin'

To retrieve both the config and state values of both then additionally the use
`*` in place of `config`:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/get.interfaceswc2.gnmi)
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path:<target: 'devicesim-1', elem:<name:'interfaces' > elem:<name:'interface' key:<key:'name' value:'*' > > elem:<name:'*'> elem:<name:'enabled' >>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
> If the device is connected and the OperationState cache is populated this returns
> 4 values - `eth1` config and state enabled values and `admin` config and
> state enabled values.

### Device read only state get
To retrieve state attributes (those defined in YANG with `config false`, non-configurable
leafs), in general there is no difference with a normal gNMI Get request.

There is however a `type` qualifier **STATE** in gNMI Get, that allows only
**STATE** values to be requested (excluding any **CONFIG** attributes. For example
to retrieve all the `STATE` values from `devicesim-1`:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/get.state.gnmi)
```bash
gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: 'devicesim-1'>, type: STATE" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

> The set of possible values for type are: `ALL`, `STATE`, `CONFIG` and `OPERATIONAL`.
> If not specified `ALL` is the default `type`.
> In onos-config there is no distinction made between `STATE` and `OPERATIONAL`
> and requesting either will get both.
> This `type` can be combined with any other proto qualifier like `elem` and `prefix`

## Northbound Delete Request via gNMI
A delete request in gNMI is done using the set request with `delete` paths instead of `update` or `replace`.
To make a gNMI Set request do delete a path, use the `gnmi_cli -set` command as in the example below:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/delete.timezone.gnmi)
```bash
gnmi_cli -address onos-config:5150 -set \
    -proto "delete: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

## Northbound Subscribe Request for Stream Notifications via gNMI
Similarly, to make a gNMI Subscribe request for streaming, use the `gnmi_cli` command as in the example below,
please note the `0` as subscription mode to indicate streaming:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/subscribe.mode0.gnmi)
```bash
gnmi_cli -address onos-config:5150 \
    -proto "subscribe:<mode: 0, prefix:<>, subscription:<path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

> This command will block until there is a change at the requested value that gets
> propagated to the underlying stream. Also as per `gnmi_cli` behaviour the updates get printed twice.

## Northbound Subscribe Once Request via gNMI
Similarly, to make a gNMI Subscribe Once request, use the `gnmi_cli` command as in the example below,
please note the `1` as subscription mode to indicate to send the response once:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/subscribe.mode1.gnmi)
```bash
gnmi_cli -address onos-config:5150 \
    -proto "subscribe:<mode: 1, prefix:<>, subscription:<path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```

> This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.

## Northbound Subscribe Poll Request via gNMI
Similarly, to make a gNMI Subscribe POLL request, use the `gnmi_cli` command as in the example below,
please note the `2` as subscription mode to indicate to send the response in a polling way every `polling_interval`
specified seconds:

[gnmi](https://github.com/onosproject/onos-config/tree/master/gnmi_cli/subscribe.mode2.gnmi)
```bash
gnmi_cli -address onos-config:5150 -polling_interval 5s \
    -proto "subscribe:<mode: 2, prefix:<>, subscription:<sample_interval: 5, path: <target: 'devicesim-1', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>" \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
> This command will fail if no value is set at that specific path. This is due to limitations of the gnmi_cli.
