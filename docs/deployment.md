# Deploying onos-config

One of the goals of the onos-config project is to provide simple deployment options
that integrate with modern technologies. Deployment configurations can be found in
the `/deployments` folder in this repository.

## Deploying on Kubernetes with Helm

[Helm] is a package manager for [Kubernetes] that allows projects to provide a
collection of templates for all the resources needed to deploy on k8s. ONOS Config
provides a Helm chart for deploying a cluster for development and testing. In the
future, this chart will be extended for production use.

### Resources

The Helm chart provides resources for deploying the config service and accessing
it over the network, both inside and outside the k8s cluster:
* `Deployment` - Provides a template for ONOS Config pods
* `ConfigMap` - Provides test configurations for the application
* `Service` - Exposes ONOS Config to other applications on the network
* `Secret` - Provides TLS certificates for end-to-end encryption
* `Ingress` - Optionally provides support for external load balancing

### Local Deployment Setup

To deploy the Helm chart locally, install [Minikube] and [Helm]. On OSX, this can be done
using [Brew]:

```bash
> brew cask install minikube
> brew install kubernetes-helm
```

You will also need VirtualBox 6.0 or higher and Docker to build and deploy an image.
* VirtualBox [installation instructions](https://www.virtualbox.org/wiki/Downloads)
* Docker [installation instructions](https://docs.docker.com/v17.12/install/)


On Linux, users have [additional options](https://kubernetes.io/docs/setup/minikube/#additional-links)
for installing local k8s clusters.

Once Minikube has been installed, start it with `minikube start`. If deploying an
[ingress] to access the service from outside the cluster, be sure to give enough
memory to the VM to run [NGINX].

```bash
> minikube start --memory=4096 --disk-size=50g --cpus=4
```

Once Minikube has been started, set your Docker environment to the Minikube Docker
daemon and build the ONOS Config image:

```bash
> eval $(minikube docker-env)
> make
```

Helm requires a special pod called Tiller to be running inside the k8s cluster for deployment
management. Before using Helm you must deploy the Tiller pod via `helm init`:

```bash
> helm init
```

Wait for the Tiller pod to transition to the ready state before proceeding:

```bash
> kubectl get pods -n kube-system
NAME                                        READY   STATUS    RESTARTS   AGE
...
tiller-deploy-659d9559f5-k4v6p              1/1     Running   0          65s
```

The onos-config Helm charts also support exposing the config service externally via an [`Ingress`] 
resource. But to support ingress, the Kubernetes cluster must first be configured with an
[ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/).
Fortunately, Minikube ships with the [NGINX] ingress controller provided as an addon, so to enable
ingress via NGINX simply enable the `ingress` addon:

```bash
> minikube addons enable ingress
```

Again, ensure the `nginx-ingress-controller-x` pod in the `kube-system` namespace is running and ready:

```bash
> kubectl get pods -n kube-system
NAME                                        READY   STATUS    RESTARTS   AGE
...
nginx-ingress-controller-586cdc477c-whbx5   1/1     Running   0          7m52s
```

### Installing the Chart

To install the chart, simply run `helm install deployments/helm/onos-config` from
the root directory of this project:

```bash
> helm install deployments/helm/onos-config
NAME:   jumpy-tortoise
LAST DEPLOYED: Tue May 14 18:56:39 2019
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1/Service
NAME                                TYPE      CLUSTER-IP    EXTERNAL-IP  PORT(S)         AGE
jumpy-tortoise-onos-config          NodePort  10.106.9.103  <none>       5150:32271/TCP  0s

==> v1/Deployment
NAME                                DESIRED  CURRENT  UP-TO-DATE  AVAILABLE  AGE
jumpy-tortoise-onos-config          1        1        1           0          0s

==> v1beta1/Ingress
NAME                                        HOSTS                   ADDRESS  PORTS  AGE
jumpy-tortoise-onos-config-ingress          config.onosproject.org  80, 443  0s

==> v1/Pod(related)
NAME                                                 READY  STATUS             RESTARTS  AGE
jumpy-tortoise-onos-config-655964cbf5-tkcfb          0/1    ContainerCreating  0         0s

==> v1/Secret
NAME                                       TYPE    DATA  AGE
jumpy-tortoise-onos-config-secret          Opaque  4     0s

==> v1/ConfigMap
NAME                                       DATA  AGE
jumpy-tortoise-onos-config-config          5     0s
```

`helm install` assigns a unique name to the chart and displays all the k8s resources that were
created by it. To list the charts that are installed and view their statuses, run `helm ls`:

```bash
> helm ls
NAME          	REVISION	UPDATED                 	STATUS  	CHART                    	APP VERSION	NAMESPACE
...
jumpy-tortoise	1       	Tue May 14 18:56:39 2019	DEPLOYED	onos-config-0.0.1	        0.0.1      	default
```

To view the pods that are deployed, run `kubectl get pods`:

```bash
> kubectl get pods
NAME                                                  READY   STATUS    RESTARTS   AGE
...
jumpy-tortoise-onos-config-655964cbf5-tkcfb           1/1     Running   0          52s
```

You can view more detailed information about the pod and other resources by running `kubectl describe`:

```bash
> kubectl describe pod jumpy-tortoise-onos-config-655964cbf5-tkcfb
Name:               jumpy-tortoise-onos-config-655964cbf5-tkcfb
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
> kubectl get svc
NAME                                        DATA   AGE
...
jumpy-tortoise-onos-config-config           5      86s
```

The application's configuration is stored in a `ConfigMap` which can be viewed by running
`kubectl get configmaps`:
```bash
> kubectl get cm
NAME                                        DATA   AGE
...
jumpy-tortoise-onos-config-config           5      97s
```

And TLS keys and certs are stored in a `Secret` resource:

```bash
> kubectl get secrets
NAME                                        TYPE                                  DATA   AGE
...
jumpy-tortoise-onos-config-secret           Opaque                                4      109s
```

#### Ingress

You can optionally enable [ingress] by overriding `ingress.enabled`. Note that you
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

| Note that the `onos` command-line client must be built or installed as described in 
the [CLI documentation](cli.md#Client).

Clients must connect through the HTTPS port using the certificates with which the ingress
was configured. Currently, the certificates used by the Helm chart can be found in the
`deployments/helm/onos-config/files/certs` directory.


### Command-line shell container

For containerized environments like Kubernetes, a Docker image `onosproject/onos-cli` is provided.
This image is built as part of the normal build.

To use the CLI in Kubernetes, run the `onosproject/onos-cli` image in a single pod deployment:

```bash
> kubectl run onos-cli --rm -it --image onosproject/onos-cli:latest --image-pull-policy "IfNotPresent" --restart "Never"
```
This command will run the CLI image as a Deployment and log into the bash shell.
Once you've joined the container, you can connect to the `onos-config` server by running:

```bash
> onos config config set address something-else-onos-config:5150
something-else-onos-config:5150
```
Note that this is only necessary if you named your deployment something else than `onos-config`.
Once the controller address is set, you should be able to execute any of the ONOS commands without 
specifying the controller address each time. See the [onos-cli](cli.md) for the full usage information.

Once the shell is exited, the Deployment will be deleted.

### Deploying the device simulator

ONOS Config provides a device simulator
for end-to-end testing. As with the onos-config app, a [Helm] chart is provided for
deployment in [Kubernetes]. Each chart instance deploys a single simulated device
`Pod` and a `Service` through which the simulator can be accessed. The onos-config chart can
then be configured to connect to the devices in k8s.

Device simulators can be deployed using the `deployments/helm/device-simulator` chart:

```bash
> helm install -n device-1 deployments/helm/device-simulator
NAME:   device-1
LAST DEPLOYED: Sun May 12 01:16:41 2019
NAMESPACE: default
STATUS: DEPLOYED

RESOURCES:
==> v1/ConfigMap
NAME                              DATA  AGE
device-1-device-simulator-config  1     1s

==> v1/Service
NAME                       TYPE       CLUSTER-IP     EXTERNAL-IP  PORT(S)    AGE
device-1-device-simulator  ClusterIP  10.110.252.69  <none>       10161/TCP  1s

==> v1/Pod
NAME                       READY  STATUS             RESTARTS  AGE
device-1-device-simulator  0/1    ContainerCreating  0         1s
```

The device-simulator chart deploys a single `Pod` containing the device simulator with a `Service`
through which it can be accessed. The device simulator's service can be seen by running the
`kubectl get services` command:

```bash
> kubectl get svc
NAME                              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
device-1-device-simulator         ClusterIP   10.106.28.52    <none>        10161/TCP        25m
```

onos-config pods can be connected to the device through this service by passing the service name
and port to the onos-config Helm chart via the `devices` option:

```bash
> helm install \
    -n onos-config \
    --set ingress.enabled=true \
    --set devices='{device-1-device-simulator}' \
    deployments/helm/onos-config
```

You can verify that the onos-config pods were connected to the device by checking the logs:

```bash
> kubectl get pods
device-1-device-simulator                          1/1     Running       0          86s
onos-config-onos-config-6f476ddc95-rgpgs           1/1     Running       0          16s
> kubectl logs onos-config-onos-config-6f476ddc95-rgpgs
2019/05/15 07:07:36 Creating Manager
2019/05/15 07:07:36 Starting Manager
2019/05/15 07:07:36 Connecting to device-1-device-simulator:10161 over gNMI
2019/05/15 07:07:36 Could not get target Client for {device-1-device-simulator:10161} does not exist, create first
2019/05/15 07:07:36 Loading default CA onfca
2019/05/15 07:07:36 Event listener initialized
2019/05/15 07:07:36 Loading default certificates
2019/05/15 07:07:36 device-1-device-simulator:10161 Connected over gNMI
2019/05/15 07:07:36 device-1-device-simulator:10161 Capabilities supported_models:<name:"openconfig-interfaces" organization:"OpenConfig working group" version:"2.0.0" > supported_models:<name:"openconfig-openflow" organization:"OpenConfig working group" version:"0.1.0" > supported_models:<name:"openconfig-platform" organization:"OpenConfig working group" version:"0.5.0" > supported_models:<name:"openconfig-system" organization:"OpenConfig working group" version:"0.2.0" > supported_encodings:JSON supported_encodings:JSON_IETF gNMI_version:"0.7.0"
```

Once onos-config has been deployed and connected to the device service, the northbound API can be
used to query and update the device configuration. This is done by setting the `-address` argument
in the `gnmi_cli` to the ingress `host:port` - i.e. `config.onosproject.org:443` - and identifying
the device in the update `target`. Devices again are identified by the name of their associated 
service and port:

```bash
> gnmi_cli -set \
    -address config.onosproject.org:443 \
    -proto "update: <path: <target: 'device-1-device-simulator', elem: <name: 'system'> elem: <name: 'clock' > elem: <name: 'config'> elem: <name: 'timezone-name'>> val: <string_val: 'Europe/Dublin'>>" \
    -timeout 5s -en PROTO \
    -client_crt deployments/helm/onos-config/files/certs/tls.crt -client_key deployments/helm/onos-config/files/certs/tls.key -ca_crt deployments/helm/onos-config/files/certs/tls.cacrt -alsologtostderr
response: <
  path: <
    elem: <
      name: "/system/clock/config/timezone-name"
    >
    target: "device-1-device-simulator"
  >
  op: UPDATE
>
timestamp: 1557904258
```

The client must connect through the HTTPS port using the certificates with which the _ingress_
was configured. The default certificates provided by the onos-config Helm chart can be found in the
`deployments/helm/onos-config/files/certs` directory.

Once you have modified the device, you can verify that onos-config handled the update successfully
by checking the onos-config logs:

```bash
> kubectl logs onos-config-onos-config-6f476ddc95-rgpgs
...
2019/05/15 07:07:36 device-1-device-simulator:10161 Connected over gNMI
2019/05/15 07:07:36 device-1-device-simulator:10161 Capabilities supported_models:<name:"openconfig-interfaces" organization:"OpenConfig working group" version:"2.0.0" > supported_models:<name:"openconfig-openflow" organization:"OpenConfig working group" version:"0.1.0" > supported_models:<name:"openconfig-platform" organization:"OpenConfig working group" version:"0.5.0" > supported_models:<name:"openconfig-system" organization:"OpenConfig working group" version:"0.2.0" > supported_encodings:JSON supported_encodings:JSON_IETF gNMI_version:"0.7.0"
2019/05/15 07:10:58 Added change M/IUW67JikD+V0clLfTEz5lTm6s= to ChangeStore (in memory)
2019/05/15 07:10:58 Change formatted to gNMI setRequest update:<path:<elem:<name:"system" > elem:<name:"clock" > elem:<name:"config" > elem:<name:"timezone-name" > > val:<string_val:"Europe/Dublin" > >
2019/05/15 07:10:58 device-1-device-simulator:10161 SetResponse response:<path:<elem:<name:"system" > elem:<name:"clock" > elem:<name:"config" > elem:<name:"timezone-name" > > op:UPDATE >
```

If the update was successful, you should be able to read the updated state of the device
through the northbound API:

```bash
> gnmi_cli -get \
    -address config.onosproject.org:443 \
    -proto "path: <target: 'device-1-device-simulator', elem: <name: 'system'> elem: <name: 'clock' > elem:<name:'config'> elem: <name: 'timezone-name'>>" \
    -timeout 5s -en PROTO -alsologtostderr \
    -client_crt deployments/helm/onos-config/files/certs/tls.crt -client_key deployments/helm/onos-config/files/certs/tls.key -ca_crt deployments/helm/onos-config/files/certs/tls.cacrt
notification: <
  timestamp: 1557856109
  update: <
    path: <
      elem: <
        name: "system"
      >
      elem: <
        name: "clock"
      >
      elem: <
        name: "config"
      >
      elem: <
        name: "timezone-name"
      >
      target: "device-1-device-simulator"
    >
    val: <
      ascii_val: "Europe/Dublin"
    >
  >
>
```

#### Deploying multiple simulators

To deploy onos-config with multiple simulators, simply install the simulator chart _n_ times
to create _n_ devices, each with a unique name:

```bash
> helm install -n device-1 deployments/helm/device-simulator
> helm install -n device-2 deployments/helm/device-simulator
> helm install -n device-3 deployments/helm/device-simulator
```

Then pass a comma-separated list of device services to configure the onos-config pods to connect
to each of the devices:

```bash
> kubectl get svc
NAME                        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)     AGE
device-1-device-simulator   ClusterIP   10.106.28.52    <none>        10161/TCP   49s
device-2-device-simulator   ClusterIP   10.98.157.220   <none>        10161/TCP   23s
device-3-device-simulator   ClusterIP   10.110.71.250   <none>        10161/TCP   17s
> helm install \
    -n onos-config \
    --set ingress.enabled=true \
    --set devices='{device-1-device-simulator,device-2-device-simulator,device-3-device-simulator}' \
    deployments/helm/onos-config
```

[Brew]: https://brew.sh/
[Helm]: https://helm.sh/
[Kubernetes]: https://kubernetes.io/
[k8s]: https://kubernetes.io/
[Minikube]: https://kubernetes.io/docs/setup/minikube/
[NGINX]: https://www.nginx.com/
[ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/

