#!/bin/bash

# Get the aggregate configuration of a device from the store

ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

# Device name
DEVICE_NAME="localhost:10161"

go run github.com/onosproject/onos-config/cmd/diags/devicetree \
    -devicename $DEVICE_NAME \
    -certPath $ONOS_CONFIG_STORE/tools/test/devicesim/certs/client1.crt \
    -keyPath $ONOS_CONFIG_STORE/tools/test/devicesim/certs/client1.key