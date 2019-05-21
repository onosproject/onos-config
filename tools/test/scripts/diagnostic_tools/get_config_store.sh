#!/bin/bash

# Get details from the Configuration store 
ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

go run github.com/onosproject/onos-config/cmd/diags/configs \
    -certPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.crt \
    -keyPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.key