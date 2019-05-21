#!/bin/bash

# Run onos-config server locally

ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

go run github.com/onosproject/onos-config/cmd/onos-config-manager \
    -configStore=$HOME/go/src/github.com/onosproject/onos-config/configs/configStore-sample.json \
    -changeStore=$HOME/go/src/github.com/onosproject/onos-config/configs/changeStore-sample.json \
    -deviceStore=$HOME/go/src/github.com/onosproject/onos-config/configs/deviceStore-sample.json \
    -networkStore=$HOME/go/src/github.com/onosproject/onos-config/configs/networkStore-sample.json
