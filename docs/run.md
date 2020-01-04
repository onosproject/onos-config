# Running onos-config 

> The commands shown below can be run from anywhere on your PC provided that go tools are installed
> and the GOPATH environment variable is set, e.g. `export GOPATH=~/go`

## Run with Helm charts
`onos-config` can run through Helm Charts as defined in the [deployment.md](deployment.md) page.
> Running with Helm is Work in Progress

## Run with `onit`
`onos-config` can run through the `onit` tool. You can find more information on how to setup `onit` in the
 [debugging](../../onos-test/docs/debugging.md) page,
 and how to run `onit` at [getting started](../../onos-test/docs/getting-started.md) page.

## Loading Model Plugins 
The model-plugin for your device can be built and loaded as outlined in the [modelplugin](modelplugin.md) guide.
> When running with Docker or Kubernetes these plugins will be built and (optionally) loaded
at startup. To check the list of currently loaded plugins use:
```bash
> onos config get plugins
```

## Northbound gNMI service
The system provides a full implementation of the gNMI spec as a northbound service.

> On a deployed cluster the onos-cli pod has a gNMI client that can be used to
> format and send gNMI messages.
To access the CLI use
```
onit onos-cli
```
to get in to the **onos-cli** pod and then run gnmi_cli from there.

Here is an example on how to use `gnmi_cli -get` to get configuration for a particular device (target) from the system.
```bash
> gnmi_cli -get -address onos-config:5150 \
    -proto "path: <target: 'localhost-1', elem: <name: 'system'> elem:<name:'config'> elem: <name: 'motd-banner'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt /etc/ssl/certs/client1.crt \
    -client_key /etc/ssl/certs/client1.key \
    -ca_crt /etc/ssl/certs/onfca.crt
```
[Full list of the gNMI northbound endpoints](gnmi.md)

## Administrative and Diagnostic Tools
The project provides enhanced northbound functionality though administrative and 
diagnostic tools, which are integrated into the consolidated `onos` command.

For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
> onos config get net-changes
```

Or, run the following to list all changes submitted through the northbound gNMI 
as they are tracked by the system broken-up into device specific batches:
```bash
> onos config get changes
```

You can read more comprehensive documentation of the various 
[administrative and diagnostic commands](cli.md).
