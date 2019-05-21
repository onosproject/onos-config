#!/bin/bash

: 'list all changes submitted through the northbound gNMI 
as they are tracked by the system broken-up into device specific batches'

ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

go run github.com/onosproject/onos-config/cmd/diags/changes \
    -certPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.crt \
    -keyPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.key
