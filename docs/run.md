<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# Running onos-config 

## Key concepts
### Internal storage
Internally `onos-config` manages configuration through `NetworkChange`s and their
child objects `DeviceChange`s. These hold all of the history of configurations
in distributed Atomix stores. They can be accessed through the Admin and Diags gRPC interface.

The gNMI interface northbound and southbound acts as a facade on top of these change objects.

### Initial synchronization of devices
`onos-config` is assumed to be the **master** of the configuration for any devices
connected to it. For this reason `onos-config` never reads configuration from a
device. This means that when `onos-config` connects to a device the first time
it does **not** synchronize the device's configuration up in to `onos-config` - if
this is required it is recommended to do it through a service above `onos-config`.

### Southbound interface
`onos-config` **only** supports a `gnmi` interface on the southbound to devices.
An adapter for connecting to NETCONF devices is [planned](https://github.com/onosproject/gnmi-netconf-adapter).
A model plugin containing the YANG models for the device, must be loaded in to
`onos-config` to allow configuration to happen.

### State attributes
Corresponding to YANG definition of **config false** some attributes on a device
are read only. These will be read from the device on connection and held in a cache.
A subscription is created to these attributes on the device and is used to keep
the cache up to date. Subscriptions on the northbound gNMI can subscribe to the
cache updates. 

## Run with Helm charts
`onos-config` can only be run on a Kubernetes cluster through Helm Charts
as defined in the [deployment.md](deployment.md) page.

## Loading Model Plugins 
The model-plugin for your device can be built and loaded as outlined in the [modelplugin](modelplugin.md) guide.
> When running with Kubernetes these plugins are loaded as "sidecar" containers
at startup, as defined in the Helm Chart.

To check the list of currently loaded plugins use:
```bash
> onos config get plugins
```

## Northbound gNMI service
The system provides a full implementation of the gNMI spec as a northbound service.

> On a deployed cluster the onos-cli pod has a gNMI client tool **gnmi_cli**
> that can be used to format and send gNMI messages.

You can run the following command to get in to the **onos-cli** pod and then run
`gnmi_cli` from there:

```bash
kubectl -n onos exec -it $(kubectl -n onos get pods -l type=cli -o name) -- /bin/sh
```

Or you can use k8s port forwarding to run gnmi_cli locally on your machine as follows:

```bash
kubectl port-forward -n <onos-namespace> <onos-config-pod-id> 5150:5150
```

For example use `gnmi_cli -capabilities` to get the capabilities from the system.

```bash
> gnmi_cli -capabilities --address=onos-config:5150 \ 
  -timeout 5s -insecure \
  -client_crt /etc/ssl/certs/client1.crt -client_key /etc/ssl/certs/client1.key -ca_crt /etc/ssl/certs/onfca.crt
```
[Full guide to the gNMI northbound endpoints](gnmi.md)

## Administrative and Diagnostic Tools
The project provides enhanced northbound functionality though administrative and 
diagnostic tools, which are integrated into the consolidated `onos` command.

For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
> onos config get network-changes
```

Or, run the following to list all changes submitted through the northbound gNMI 
as they are tracked by the system broken-up into device specific batches:
```bash
> onos config get device-changes <device-name>
```

You can read more comprehensive documentation of the various 
[administrative and diagnostic commands](cli.md).
