#!/bin/bash

# Make a gNMI subscribe request  

ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config
# Mode defines the subscription type (0=STREAM, 1=ONCE,2=POLL)
MODE=1

# The request message
PROTO="subscribe:<mode:"$MODE", prefix:<>, subscription:<path: <target: 'localhost:10161', elem: <name: 'openconfig-system:system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>>>>"

# Time out value
TIME_OUT=5s


gnmi_cli -address localhost:5150 \
    -proto "$PROTO" \
    -timeout $TIME_OUT  \
    -client_crt $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.crt \
    -client_key $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.key \
    -ca_crt $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/onfca.crt \
    -alsologtostderr
