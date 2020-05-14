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
  add             Add a config resource
  compact-changes Takes a snapshot of network and device changes
  config          Manage the CLI configuration
  get             Get config resources
  load            Load configuration from a file
  rollback        Rolls-back a network change
  snapshot        Commands for managing snapshots
  watch           Watch for updates to a config resource type

Flags:
  -h, --help                     help for config
      --no-tls                   if present, do not use TLS
      --service-address string   the onos-config service address (default "onos-config:5150")
      --tls-cert-path string     the path to the TLS certificate
      --tls-key-path string      the path to the TLS key

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

### Loading configuration data in bulk
Configuration data can be loaded in to onos-config through the cli with
```bash
onos config load yaml <filename(s)>
```

The Yaml file must be in the form below. Several updates, replace or delete entries can be made.

A separate NetworkChange will be created for each file given.
> This allows the set of updates to be broken up in to smaller groups.

```yaml
setrequest:
    prefix:
        elem:
            - name: e2node
              key: {}
        target: ""
    delete: []
    replace: []
    update:
        - path:
              element: []
              origin: ""
              elem:
                  - name: intervals
                    key: {}
                  - name: RadioMeasReportPerUe
                    key: {}
              target: 315010-0001420
          val:
              stringvalue: null
              intvalue: null
              uintvalue:
                  uintval: 20
              boolvalue: null
              bytesvalue: null
              floatvalue: null
              decimalvalue: null
              leaflistvalue: null
              anyvalue: null
              jsonvalue: null
              jsonietfvalue: null
              asciivalue: null
              protobytes: null
          duplicates: 0
```
