#!/bin/bash

# Make a gNMI Get request

# Path to onos-config root directory
ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

# The request message
PROTO="path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem:<name:'config'> elem: <name: 'motd-banner'>>"

# Time out value
TIME_OUT=5s

gnmi_cli -get -address localhost:5150 \
    -proto "$PROTO"  \
    -timeout $TIME_OUT  \
    -client_crt $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.crt \
    -client_key $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.key \
    -ca_crt $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr 
