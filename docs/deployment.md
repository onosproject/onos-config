<!--
SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>

SPDX-License-Identifier: Apache-2.0
-->

# Deploying onos-config

This guide deploy `onos-config` through it's [Helm] chart assumes you have a [Kubernetes] cluster running 
with an atomix controller deployed in a namespace.  
`onos-config` Helm chart is based on Helm 3.0 version, with no need for the Tiller pod to be present.   
If you don't have a cluster running and want to try on your local machine please follow first 
the [Kubernetes] setup steps outlined to [deploy with Helm](https://docs.onosproject.org/developers/deploy_with_helm/).
The following steps assume you have the setup outlined in that page, including the `micro-onos` namespace configured. 

## Installing the Chart

To install the chart in the `micro-onos` namespace run from the root directory of the `onos-helm-charts` repo the command:
```bash
helm install -n micro-onos onos-config onos-config
```
The output should be:
```bash
NAME: onos-config
LAST DEPLOYED: Tue Nov 26 13:38:20 2019
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

`helm install` assigns a unique name to the chart and displays all the k8s resources that were
created by it. To list the charts that are installed and view their statuses, run `helm ls`:

```bash
helm ls
NAME          	REVISION	UPDATED                 	STATUS  	CHART                    	APP VERSION	NAMESPACE
...
jumpy-tortoise	1       	Tue May 14 18:56:39 2019	DEPLOYED	onos-config-0.0.1	        0.0.1      	default
```

### Onos Config Deployment

```bash
NAME                                 READY   STATUS    RESTARTS   AGE
atomix-controller-6bb9555f48-6qckx   1/1     Running   0          18h
device-simulator-597559fdd4-s6z8w    1/1     Running   0          14h
onos-cli-5ffc6748cc-zfm74            1/1     Running   0          18h
onos-config-85d6bd66b4-jmjnx         5/5     Running   0          18h
onos-config-consensus-1-0            1/1     Running   0          18h
onos-gui-6f849cc8f5-mth2s            2/2     Running   0          18h
onos-topo-757855bcf-z7vwn            1/1     Running   0          18h
onos-topo-consensus-1-0              1/1     Running   0          18h
```

> `onos-config` is dependent on the `onos-topo` service which has been started here
> through its own Helm chart. Also a `device-simulator` has been started - it
> supports the gNMI interface and is configured with the **Devicesim-1.0.0** model
>
> Note here that `onos-config` is showing 5 containers in the pod - 1 for `onos-config`
> itself and 4 model plugins have been loaded. In addition another pod "consensus"
> is running to connect to the "atomix-controller".  

One can customize the number of partitions and replicas by modifying, in `values.yaml` 
the values of:
```bash 
store.consensus.partitions: 1
store.consensus.backend.replicas: 1
```

### Installing the chart in a different namespace.

Issue the `helm install` command substituting `micro-onos` with your namespace.
```bash
helm install -n <your_name_space> onos-config onos-config
```
### Installing the chart with debug. 
`onos-config` offers the capability to open a debug port (4000) to the image.
To enable the debug capabilities please set the debug flag to true in `values.yaml` or pass it to `helm install`
```bash
helm install -n micro-onos onos-config onos-config --set debug=true
```

### Troubleshoot

If your chart does not install or the pod is not running for some reason and/or you modified values Helm offers two flags to help you
debug your chart: 

* `--dry-run` check the chart without actually installing the pod. 
* `--debug` prints out more information about your chart

```bash
helm install -n micro-onos onos-config --debug --dry-run onos-topo
```
Also to verify how template values are expanded, run:
```bash
helm install template onos-config
```

## Uninstalling the chart

To remove the `onos-config` pod issue
```bash
 helm delete -n micro-onos onos-config
```

## Including configuration model plugins
To include specific configuration model plugins as part of the `onos-config` pod,
list their names, their image name and the desired gRPC port in the `modelPlugins`
section of the `values.yaml` file of the `onos-config` chart. See below:

```yaml
modelPlugins:
  - name: devicesim-1
    image: onosproject/devicesim:0.5.5-devicesim-1.0.0
    endpoint: localhost
    port: 5152
  - name: testdevice-1
    image: onosproject/testdevice:0.5.5-testdevice-1.0.0
    endpoint: localhost
    port: 5153
  - name: testdevice-2
    image: onosproject/testdevice:0.5.5-testdevice-2.0.0
    endpoint: localhost
    port: 5154
```
This value map will be processed during deployment time and will start each given
model plugin as a sidecar in the `onos-config` pod, with the appropriate gRPC port number.

## Pod Information

To view the pods that are deployed, run `kubectl -n micro-onos get pods`:

```bash
> kubectl -n micro-onos get pods
NAME                                                  READY   STATUS    RESTARTS   AGE
...
onos-config-655964cbf5-tkcfb           1/1     Running   0          52s
```

You can view more detailed information about the pod and other resources by running `kubectl describe`:

```bash
> kubectl -n micro-onos describe pod onos-config-655964cbf5-tkcfb
Name:               onos-config-655964cbf5-tkcfb
Namespace:          default
Priority:           0
PriorityClassName:  <none>
Node:               minikube/10.0.2.15
Start Time:         Tue, 14 May 2019 18:56:39 -0700
...
```

The onos-config pods are reached through a `Service` which load balances requests to the application.
To view the services, run `kubectl get services`:

```bash
> kubectl -n micro-onos get svc
NAME                                        DATA   AGE
...
onos-config-config           5      86s
```

The application's configuration is stored in a `ConfigMap` which can be viewed by running
`kubectl get configmaps`:
```bash
> kubectl -n micro-onos get cm
NAME                                        DATA   AGE
...
onos-config-config           5      97s
```

And TLS keys and certs are stored in a `Secret` resource:

```bash
> kubectl -n micro-onos get secrets
NAME                                        TYPE                                  DATA   AGE
...
onos-config-secret           Opaque                                4      109s
```

[Brew]: https://brew.sh/
[Helm]: https://helm.sh/
[Kubernetes]: https://kubernetes.io/
[k8s]: https://kubernetes.io/
[kind]: https://kind.sigs.k8s.io
[NGINX]: https://www.nginx.com/
[ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/

