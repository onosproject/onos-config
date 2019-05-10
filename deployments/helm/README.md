## ONOS Config Helm Chart

### Local deployment

To deploy the Helm chart locally, install [Minikube] and [Helm]. On OSX, this can be done
using [Brew]:

```bash
> brew install minikube
> brew install helm
```

Start Minikube with `minikube start`. If deploying an [ingress] to access the service
from outside the cluster, be sure to give enough memory to the VM to run NGINX.

```bash
> minikube start --memory=4096 --disk-size=50g --cpus=4
```

Once Minikube has been started, set your Docker environment to the Minikube Docker
daemon and build the ONOS Config image:

```bash
> eval $(minikube docker-env)
> make
```

Initialize Tiller to enable Helm for your cluster:

```bash
> helm init
```

The [NGINX] ingress controller is shipped with later versions of Minikube. If you
need ingress support, simply enable the ingress addon:

```bash
> minikube addons enable ingress
```

### Installing the chart

To install the chart, run `helm install deployments/helm` from the root
directory of this project:

```bash
> helm install deployments/helm
```

You can optionally enable [ingress] by overriding `ingress.enabled`. Note you must
have an ingress controller installed:

```bash
> helm install \
    -n onos-config \
    --set ingress.enabled=true \
    deployments/helm
```

```bash
> go run cmd/admin/client/main.go \
    -address=10.0.2.15:443 \
    -host="config.onosproject.org" \
    -keyPath=deployments/helm/files/certs/tls.key \
    -certPath=deployments/helm/files/certs/tls.crt
```

[Brew]: https://brew.sh/
[Helm]: https://helm.sh/
[Minikube]: https://kubernetes.io/docs/setup/minikube/
[NGINX]: https://www.nginx.com/
[ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
