# Administrative and Diagnostic Command-Line
The project provides a command-line facilities for remotely 
interacting with the administrative and diagnostic services of the `onos-config` server.

The commands are available at run-time using the consolidated `onos` client hosted in 
the `onos-cli` repository, but their implementation is hosted and built here.

The documentation about building and deploying the consolidate `onos` client or its Docker container
is available in the `onos-cli` GitHub repository.

## Usage
```bash
> onos config --help
ONOS configuration subsystem commands

Usage:
  onos config [command]

Available Commands:
  add         Add a config resource
  get         Get config resources
  rollback    Rolls-back a network configuration change
  watch       Watch for updates to a config resource type

Flags:
  -h, --help   help for config

Use "onos config [command] --help" for more information about a command.
```

### Global Flags
Since the `onos` command is a client, it requires the address of the server as well
as the paths to the key and the certificate to establish secure connection to the 
server.

These options are global to all commands and can be persisted to avoid having to
specify them for each command. For example, you can set the default server address
as follows:
```bash
> onos config config set address onos-config-server:5150
```

Subsequent usages of the `onos` command can then abstain from using the `--address` 
option to indicate the server address, resulting in easier usage.

## Example Commands

### Rollback Network Change
To rollback a network use the rollback admin tool. This will rollback the last network
change unless a specific change is given with the `changename` parameter
```bash
> onos config rollback Change-VgUAZI928B644v/2XQ0n24x0SjA=
```

### Listing and Loading model plugins
A model plugin is a shared object library that represents the YANG models of a
particular Device Type and Version. The plugin allows user to create and load
their own device models in to onos-config that can be used for validating that
configuration changes observe the structure of the YANG models in use on the
device. This improves usability by pushing information about the devices'
model back up to the onos-config gNMI northbound interface.

Model plugins can be loaded at the startup of onos-config by (repeated) `--modelPlugin`
options, or they can be loaded at run time. To see the list of currently loaded
plugins use the command:
```bash
> onos config get plugins
```

To load a plugin dynamically at runtime use the command:
```bash
> onos config add plugin <full path and filename of a compatible shared object library on target machine>
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

### List and Watch Changes
For example, to list and watch all changes stored internally run:
```bash
> onos config watch network-changes
...
```
or to watch `device-changes`
```bash
> onos config watch device-changes
...
```

### Devicetree
To get the aggregate configuration of a device in a hierarchical JSON structure from the store use:
```bash
> onos config get devicetree --layer 0 Device1
DEVICE			CONFIGURATION		TYPE		VERSION
Device1                 Device1-1.0.0           TestDevice      1.0.0
CHANGE:	2uUbeEV4i3ADedjeORmgQt6CVDM=
CHANGE:	tAk3GZSh1qbdhdm5414r46RLvqw=
CHANGE:	MY8s8Opw+xjbcARIMzIpUIzeXv0=
TREE:
{"cont1a":{"cont2a":{"leaf2a":13,"leaf2b":1.14159,"leaf2c":"def","leaf2d":0.002,"leaf2e":[-99,-4,5,200],"leaf2g":false},"leaf1a":"abcdef","list2a":[{"name":"txout1","tx-power":8},{"name":"txout3","tx-power":16}]},"test1:leafAtTopLevel":"WXY-1234"}
```

> This displays the list of changes IDs and the aggregate effect of layering each
> one on top of the other. This is the _effective_ configuration.
>
> By default all layers are shown (`layer=0`). To show the previous _effective_
> configuration use `layer=-1`
>
> To display the devices trees for all devices, just omit the device name.
