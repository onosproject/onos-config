# Administrative Tools
The project provides a number of administrative tools for remotely accessing the enhanced northbound
functionality.

### List Network Changes
For example, to list all network changes submitted through the northbound gNMI interface run:
```bash
go run github.com/onosproject/onos-config/cmd/admin/net-changes
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
go run github.com/onosproject/onos-config/cmd/admin/rollback \
    -changename Change-VgUAZI928B644v/2XQ0n24x0SjA=
```