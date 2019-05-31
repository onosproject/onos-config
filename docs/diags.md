# Diagnostic Tools
There are a number of commands that provide internal view into the state the onos-config store.
These tools use a special-purpose gRPC interfaces to obtain the internal meta-data
from the running onos-config process. Please note that these tools are intended purely for
diagnostics and should not be relied upon for programmatic purposes as they are not subject
to any backward compatibility guarantees.

For example, run the following to list all changes submitted through the northbound gNMI 
as they are tracked by the system broken-up into device specific batches:
```bash
go run github.com/onosproject/onos-config/cmd/diags/changes
```
> For a specific change use the -changeid argument


To get details from the Configuration store use
```bash
go run github.com/onosproject/onos-config/cmd/diags/configs
```
> For the configuration for a specific device use the -devicename argument


To get the aggregate configuration of a device from the store use
```bash
go run github.com/onosproject/onos-config/cmd/diags/devicetree \
    -devicename localhost:10161
```

> Of course, there will be many more such commands available in the near future
