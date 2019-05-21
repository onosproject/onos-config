#!/bin/bash

# list all network changes submitted through the northbound gNMI interface run'

ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

go run github.com/onosproject/onos-config/cmd/admin/net-changes \
    -certPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.crt \
    -keyPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.key