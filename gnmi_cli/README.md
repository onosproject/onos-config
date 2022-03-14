<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

The GNMI CLI may be used to demo the `onos-config`. Example files in this directory.

See more details at [https://docs.onosproject.org/onos-config/docs/gnmi/](https://docs.onosproject.org/onos-config/docs/gnmi/)

Typical usage for **get** is as follows:
```
gnmi_cli -get -address localhost:5150 \
    -timeout 5s -en PROTO -alsologtostderr -insecure \
    -client_crt ../onos-helm-charts/onos-config/files/certs/tls.crt -client_key ../onos-helm-charts/onos-config/files/certs/tls.key -ca_crt ../onos-helm-charts/onos-config/files/certs/tls.cacrt \
    -proto "$(cat gnmi_cli/get.timezone.gnmi)"
```

> This assumes that
> 1. there is a **port-forward** in place on port 5150 to the `onos-config`.
> instance deployed on Kubernetes, making onos-config gRPC available at `localhost:5150`.
> 1. your working directory is from a clone of the `onos-config` git repo.
> 1. that you have installed `gnmi_cli` locally.
> 1. that there is a device-simulator created on the K8S cluster
> 1. that there is a device `devicesim-1` created in 'onos-topo'. `helm -n micro-onos install device-simulator device-simulator`
> done on the onos cli with `onos topo add device devicesim-1 --address device-simulator:11161 --version 1.0.0 --type Devicesim --plain`
> 1. that you have checked the connectivity with `onos config get opstate devicesim-1`

Similar calls can be made for **-set**, **-delete**, **-subscribe** etc.

In addition the `gnmi_cli` can also be called directly on the device at
"device-simulator:11161" (which is a plain connection - use `gnmi_cli -tlsDisabled` -
only available on the ONF [fork](https://github.com/opennetworkinglab/gnmi) )
