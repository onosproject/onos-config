<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# Ingress for onos-config

---
**NOTE**

This file has to be revisited, please have no expectation of correctness.

---

In the `onos-config` helm chart you can optionally enable [ingress] by overriding `ingress.enabled`. Note that you
must have an ingress controller installed/enabled as described above:

```bash
> helm install \
    -n onos-config \
    --set ingress.enabled=true \
    deployments/helm/onos-config
```

By default, the ingress controller uses the self-signed certificates that ship with the 
chart to provide end-to-end routing, load balancing, and encryption, making the onos-config
services accessible from outside the k8s cluster. The default certificates expect the
service to be reached through the `config.onosproject.org` domain. Thus, to connect
to the service through the ingress, you must configure `/etc/hosts` to point to the
load balancer's IP:

```bash
192.168.99.102 config.onosproject.org
```

The IP address of the ingress may differ depending on the environment. In clustered environments,
the ingress IP is typically read from the `ingress` resource:

```bash
> kubectl get ingress
NAME                                      HOSTS                    ADDRESS     PORTS     AGE
onos-config-onos-config-ingress           config.onosproject.org   10.0.2.15   80, 443   76m
```

However, since Minikube runs in a VM, the ingress must be reached through the Minikube VM's IP
which can be found via the `minikube ip` command:

```bash
LBIP=$(minikube ip)
```

In clustered environments, the ingress IP can be retrieved from the ingress
metadata:

```bash
> kubectl get ingress
NAME                                      HOSTS                    ADDRESS     PORTS     AGE
onos-config-onos-config-ingress           config.onosproject.org   10.0.2.15   80, 443   76m
```

Once you've located the ingress IP address and configured `/etc/hosts`, you can connect to
the onos-config service via the ingress load balancer:

```bash
> onos config --address=config.onosproject.org:443 get changes
```

[ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/