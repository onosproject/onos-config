# Administrative Tools
The project provides a number of administrative tools for remotely accessing the enhanced northbound
functionality.

### List Network Changes
For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
> go run github.com/onosproject/onos-config/cmd/admin/net-changes
```
> While this tool (and all the other utilities listed below) has the option to
> specify a -keyPath and -certPath for a client certificate to make the connection
> to the gRPC admin interface, those arguments can be omitted at runtime, leaving
> the internal client key and cert at 
> [default-certificates.go](../pkg/certs/default-certificates.go) to be used.

### Rollback Network Change
To rollback a network use the rollback admin tool. This will rollback the last network
change unless a specific change is given with the **-changename** parameter
```bash
> go run github.com/onosproject/onos-config/cmd/admin/rollback \
    -changename Change-VgUAZI928B644v/2XQ0n24x0SjA=
```

### Adding, Removing and Listing Devices
Until the full topology subsystem is available, there is a provisional 
administrative interface that allows devices to be added, removed and listed via gRPC.
A command has been provided to allow manipulating the device inventory from the command
line using this gRPC service.

To add a new device, specify the device informatio protobuf encoding as the value of the 
`addDevice` option. The 'id', 'address' and 'version' fields are required at the minimum.
For example:

```bash
> go run github.com/onosproject/onos-config/cmd/admin/devices --addDevice \
        "id: 'device-4', address: 'localhost:10164' version: '1.0.0'"
Adding device device-4
```

In order to remove a device, specify its ID as a value to the `removeID` option as follows:
```bash
> go run github.com/onosproject/onos-config/cmd/admin/devices --removeID device-2 
Removing device device-2
```

If you do not specify any options, the command will list all the devices currently in the inventory:
```bash
> go run github.com/onosproject/onos-config/cmd/admin/devices
device-1: localhost:10161 (1.0.0)
device-3: localhost:10163 (1.0.0)
device-4: localhost:10164 (1.0.0)
```

