#!/bin/bash

: 'rollback the last network change unless a 
   specific change is given with the -changename parameter'

ONOS_CONFIG_ROOT=$HOME/go/src/github.com/onosproject/onos-config

go run github.com/onosproject/onos-config/cmd/admin/rollback \
    -changename Change-enLNLxyL7dDL0IlLXeSj1m5PPAY=  \
    -certPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.crt \
    -keyPath $ONOS_CONFIG_ROOT/tools/test/devicesim/certs/client1.key