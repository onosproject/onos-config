<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# Administrative Command-Line
The project provides a command-line facilities for remotely 
interacting with the administrative service of the `onos-config` server.

The commands are available at run-time using the consolidated `onos` client hosted in 
the `onos-cli` repository, but their implementation is hosted and built here.

The documentation about building and deploying the consolidated `onos` command-line client or its Docker container
is available in the `onos-cli` GitHub repository.

## Usage
```bash
> onos config --help
ONOS configuration subsystem commands

Usage:
  onos config [command]

Available Commands:
  config      Manage the CLI configuration
  get         Get config resources
  log         logging api commands
  rollback    Rolls-back a transaction
  watch       Watch for updates to a config resource type

Flags:
      --auth-header string       Auth header in the form 'Bearer <base64>'
      -h, --help                 help for config
      --no-tls                   if present, do not use TLS
      --service-address string   the gRPC endpoint (default "onos-config:5150")
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

### Listing of configuration model plugins
A configuration model plugin is a sidecar container that runs in the same pod as `onos-config`.
Each plugin represents a complete set of YANG models supported by a particular device type. 
Each plugin has a name (sometimes referred to as type) and a version. 

To see the list of currently loaded plugins use the command:
```bash
> onos config get plugins
ID                  STATUS    ENDPOINT          INFO.NAME     INFO.VERSION    ERROR
testdevice-2.0.0    Loaded    localhost:5154    testdevice    2.0.0
devicesim-1.0.0     Loaded    localhost:5152    devicesim     1.0.0
testdevice-1.0.0    Loaded    localhost:5153    testdevice    1.0.0
```
See more information on building and deploying configuration model plugins in `config-models` repository.

### List configuration transactions
Configuration changes made through the `onos-config` gNMI interface are represented as _transactions_. To view the list
of these transactions use the following command:
```shell
> onos config get transactions
ID                                           INDEX    STATUS.STATE    TRANSACTIONTYPE    CREATED                                  UPDATED                                  DELETED    USERNAME    TRANSACTIONSTRATEGY.ISOLATION    TRANSACTIONSTRATEGY.SYNCHRONICITY
uuid:1865ad96-6159-4050-b9cf-7d1f7ece54c4    1        APPLIED         Change             2022-02-23 17:18:21.8499168 +0000 UTC    2022-02-23 17:18:22.0512956 +0000 UTC    <nil>                  DEFAULT                          SYNCHRONOUS
uuid:9acdbe9e-f8c3-4797-b77e-d4a324270564    2        APPLIED         Change             2022-02-23 17:19:07.1359094 +0000 UTC    2022-02-23 17:19:07.4341523 +0000 UTC    <nil>                  DEFAULT                          SYNCHRONOUS
uuid:745d9944-49aa-441f-8394-ad2e4ad70543    3        APPLIED         Change             2022-02-23 17:19:37.6194301 +0000 UTC    2022-02-23 17:20:16.6840849 +0000 UTC    <nil>                  DEFAULT                          ASYNCHRONOUS
uuid:e7df0125-3cca-4fab-a002-04ee3a67ef99    4        APPLIED         Change             2022-02-23 17:20:31.5019776 +0000 UTC    2022-02-23 17:20:31.6836693 +0000 UTC    <nil>                  DEFAULT                          SYNCHRONOUS
uuid:63bcdfd8-b9bc-4cfa-a20d-93159cd384f5    5        APPLIED         Rollback           2022-02-23 17:20:31.7456597 +0000 UTC    2022-02-23 17:20:31.9213865 +0000 UTC    <nil>                  DEFAULT                          SYNCHRONOUS
uuid:8edb7b9d-4571-4fac-9f4c-d876c0d4cafe    6        APPLIED         Change             2022-02-23 17:20:46.6952989 +0000 UTC    2022-02-23 17:20:46.8966023 +0000 UTC    <nil>                  DEFAULT                          SYNCHRONOUS
uuid:734cfad1-9154-42c3-9e73-d6bec1101267    7        APPLIED         Change             2022-02-23 17:20:46.9112378 +0000 UTC    2022-02-23 17:20:47.0962395 +0000 UTC    <nil>                  DEFAULT                          SYNCHRONOUS
uuid:095a064a-7f37-457a-933d-b991029127da    8        APPLIED         Change             2022-02-23 17:21:02.2661133 +0000 UTC    2022-02-23 17:21:02.4532263 +0000 UTC    <nil>                  DEFAULT                          SYNCHRONOUS
```

To continuously monitor the transaction events, you can use a similar command `onos config watch transactions`.

### Rollback Network Change
To rollback the most recent transaction, and revert the configuration of all targets involved in that transaction to their
prior state, use the `rollback` command and specify the `Index` of the most recent transaction.
```bash
> onos config rollback 8
```

### Listing target configurations
To list the status of all configurable targets use the following command:
```onos config get configurations
ID                                 TARGETID                           STATUS.STATE    INDEX
square-stingray                    square-stingray                    SYNCHRONIZED    39
fond-bulldog                       fond-bulldog                       SYNCHRONIZED    47
reincarnated-target                reincarnated-target                SYNCHRONIZED    3
guiding-whale                      guiding-whale                      SYNCHRONIZED    0
expert-tortoise                    expert-tortoise                    SYNCHRONIZED    33
gentle-polliwog                    gentle-polliwog                    SYNCHRONIZED    44
new-bull                           new-bull                           SYNCHRONIZED    1
```

Similarly, to continuously monitor ongoing changes to configurations, you can use the `onos config watch configurations`.

To get details on the current configuration for a specific target, use:
```onos config get configuration square-stingray -v
ID                 TARGETID           STATUS.STATE    INDEX    VALUES
square-stingray    square-stingray    SYNCHRONIZED    39       map[/system/config/login-banner:path:"/system/config/login-banner" value:<bytes:"2" type:STRING >  /system/config/motd-banner:path:"/system/config/motd-banner" value:<bytes:"1" type:STRING > ]
```
