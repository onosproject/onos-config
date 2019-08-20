# Running onos-config 

> The commands shown below can be run from anywhere on your PC provided that go tools are installed
> and the GOPATH environment variable is set, e.g. `export GOPATH=~/go`

## Run with Helm charts
`onos-config` can run through Helm Charts as defined in the [deplooyment.md](deployment.md) page.
> Running with Helm is Work in Progress

## Run with `onit`
`onos-config` can run through the `onit` tool. You can find more information on how to setup `onit` in the
 [setup.md](https://github.com/onosproject/onos-test/blob/master/docs/setup.md) page,
 and how to run `onit` at [run.md](https://github.com/onosproject/onos-test/blob/master/docs/run.md)

## Loading Model Plugins 
The model-plugin for your device can be built and loaded as outlined in the [modelplugins.md](modelplugins.md) guide.
> When running with Docker or Kubernetes these plugins will be built and (optionally) loaded
at startup. To check the list of currently loaded plugins use:
```bash
> onos config get plugins
```

## Northbound gNMI service
The system provides a full implementation of the gNMI spec as a northbound service.

Here is an example on how to use `gnmi_cli -get` to get configuration for a particular device (target) from the system.
```bash
> gnmi_cli -get -address localhost:5150 \
    -proto "path: <target: 'localhost-1', elem: <name: 'system'> elem:<name:'config'> elem: <name: 'motd-banner'>>" \
    -timeout 5s -alsologtostderr \
    -client_crt pkg/southbound/testdata/client1.crt \
    -client_key pkg/southbound/testdata/client1.key \
    -ca_crt pkg/southbound/testdata/onfca.crt
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
