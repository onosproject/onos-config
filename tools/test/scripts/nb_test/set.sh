#!/bin/bash

# Make a gNMI Set Request

# Path to onos-config root
ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

# The request message
PROTO="update: <path: <target: 'localhost:10161', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>> val: <string_val: 'Europe/Dublin'>>" 

# Time out value
TIME_OUT=5s

gnmi_cli -address localhost:5150 -set \
    -proto  "$PROTO" \
    -timeout $TIME_OUT \
    -client_crt $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.crt \
    -client_key $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.key \
    -ca_crt $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr 
    
