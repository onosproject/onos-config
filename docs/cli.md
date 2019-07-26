# Administrative and Diagnostic Command-Line
The project provides a command-line client for remotely 
interacting with the administrative and diagnostic services of the `onos-config` server.

## Client
To build or get the client into an executable, simply run:
```bash
> go run github.com/onosproject/onos-config/cmd/onos
```
or
```bash
> go get github.com/onosproject/onos-config/cmd/onos
```

### Bash or Zsh Auto-Completion
The `onos` client supports shell auto-completion for its various
commands, sub-commands and flags. To enable this, run the following from the shell:
```bash
> eval "$(onos completion bash)"
```
After that, you should be able to use the `TAB` key to obtain suggestions for 
valid options.

### Global Flags
Since the `onos` command is a client, it requires the address of the server as well
as the paths to the key and the certificate to establish secure connection to the 
server.

These options are global to all commands and can be persisted to avoid having to
specify them for each command. For example, you can set the default server address
as follows:
```bash
> onos config set address onos-config-server:5150
```

Subsequent usages of the `onos` command can then abstain from using the `--address` 
option to indicate the server address, resulting in easier usage.

### Usage
Information on the general usage can be obtained by using the `--help` flag as follows:
```bash
> onos --help
ONOS command line client

Usage:
  onos [command]

Available Commands:
  changes     Lists records of configuration changes
  completion  Generated bash or zsh auto-completion script
  config      Read and update CLI configuration options
  configs     Lists details of device configuration changes
  devices     Manages inventory of network devices
  devicetree  Lists devices and their configuration in tree format
  help        Help about any command
  init        Initialize the ONOS CLI configuration
  models      Manages model plugins
  net-changes Lists network configuration changes
  rollback    Rolls-back a network configuration change


Flags:
  -a, --address string    the controller address (default ":5150")
  -c, --certPath string   path to client certificate (default "client1.crt")
      --config string     config file (default: $HOME/.onos/config.yaml)
  -h, --help              help for onos
  -k, --keyPath string    path to client private key (default "client1.key")

Use "onos [command] --help" for more information about a command.
```

> While this tool (and all the other utilities listed below) has the option to
> specify a --keyPath and --certPath for a client certificate to make the connection
> to the gRPC interface, those arguments can be omitted at runtime, leaving
> the internal client key and cert at 
> [default-certificates.go](../pkg/certs/default-certificates.go) to be used.


## Example Commands

### List Network Changes
For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
> onos net-changes
...
```

### Rollback Network Change
To rollback a network use the rollback admin tool. This will rollback the last network
change unless a specific change is given with the `changename` parameter
```bash
> onos rollback Change-VgUAZI928B644v/2XQ0n24x0SjA=
```

### Adding, Removing and Listing Devices
Until the full topology subsystem is available, there is a provisional 
administrative interface that allows devices to be added, removed and listed via gRPC.
A command has been provided to allow manipulating the device inventory from the command
line using this gRPC service.

To add a new device, specify the device information protobuf encoding as the value of the 
`addDevice` option. The `id`, `address` and `version` fields are required at the minimum.
For example:

```bash
> onos devices add "id: 'device-4', address: 'localhost:10164' version: '1.0.0', devicetype: 'Devicesim'"
Added device device-4
```

In order to remove a device, specify its ID as follows:
```bash
> onos devices remove device-2 
Removed device device-2
```

If you do not specify any options, the command will list all the devices currently in the inventory:
```bash
> onos devices list -v
NAME			ADDRESS			VERSION
localhost-3             localhost:10163         1.0.0
	USER		PASSWORD	TIMEOUT	PLAIN	INSECURE
	                                5       false	false

stratum-sim-1           localhost:50001         1.0.0
	USER		PASSWORD	TIMEOUT	PLAIN	INSECURE
	                                5       true	false

localhost-1             localhost:10161         1.0.0
	USER		PASSWORD	TIMEOUT	PLAIN	INSECURE
	devicesim       notused         5       false	false

localhost-2             localhost:10162         1.0.0
	USER		PASSWORD	TIMEOUT	PLAIN	INSECURE
	                                5       false	false
```

### Listing and Loading model plugins
A model plugin is a shared object library that represents the YANG models of a
particular Device Type and Version. The plugin allows user to create and load
their own device models in to onos-config that can be used for validating that
configuration changes observe the structure of the YANG models in use on the
device. This improves usability by pushing information about the devices'
model back up to the onos-config gNMI northbound interface.

Model plugins can be loaded at the startup of onos-config by (repeated) --modelPlugin
options, or they can be loaded at run time. To see the list of currently loaded
plugins use the command:
```bash
> onos models list
```

To load a plugin dynamically at runtime use the command:
```bash
> onos models load <full path and filename of a compatible shared object library on target machine>
```
> NOTE: Model Plugins cannot be dynamically unloaded - a restart of onos-config
> is required to unload.
> In a distributed environment the ModelPlugin will have to be loaded on all
> instances of onos-config

## Other Diagnostic Commands
There are a number of commands that provide internal view into the state the onos-config store.
These tools use a special-purpose gRPC interfaces to obtain the internal meta-data
from the running onos-config process. Please note that these tools are intended purely for
diagnostics and should not be relied upon for programmatic purposes as they are not subject
to any backward compatibility guarantees.

### Changes
For example, run the following to list all changes submitted through the northbound gNMI 
as they are tracked by the system broken-up into device specific batches:
```bash
> onos changes
...
```
> For a specific change specify the optional `changeId` argument.

### Configs
To get details from the Configuration store use
```bash
> onos configs
...
```
> For the configuration for a specific device use the optional `deviceId` argument.

### Devicetree
To get the aggregate configuration of a device in a hierarchical JSON structure from the store use:
```bash
> onos devicetree --layer 0 Device1
DEVICE			CONFIGURATION		TYPE		VERSION
Device1                 Device1-1.0.0           TestDevice      1.0.0
CHANGE:	2uUbeEV4i3ADedjeORmgQt6CVDM=
CHANGE:	tAk3GZSh1qbdhdm5414r46RLvqw=
CHANGE:	MY8s8Opw+xjbcARIMzIpUIzeXv0=
TREE:
{"cont1a":{"cont2a":{"leaf2a":13,"leaf2b":1.14159,"leaf2c":"def","leaf2d":0.002,"leaf2e":[-99,-4,5,200],"leaf2g":false},"leaf1a":"abcdef","list2a":[{"name":"txout1","tx-power":8},{"name":"txout3","tx-power":16}]},"test1:leafAtTopLevel":"WXY-1234"}
```

> This displays the list of changes IDs and the aggregate effect of layering each
> one on top of the other. This is **effective** configuration.

> By default all layers are shown (**layer=0**). To show the previous **effective**
> configuration use **layer=-1**

> To display the devices trees for all devices, just omit the device name.
