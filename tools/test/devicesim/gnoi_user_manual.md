# gNOI Simulator User Manual

This is a docker VM that runs gNOI implementation supporting the following services:

* *Certificate management* that can be used for certificate installation and rotation. 

## Create the docker image
```bash
docker build -t devicesim  -f Dockerfile .
```

## Run 3 devices with Docker Compose
Run 3 devices with 
```bash
docker-compose -f docker-compose-gnoi.yml up
```

This gives an output like:
```bash
Creating devicesim_devicesim3_1 ... 
Creating devicesim_devicesim1_1 ... 
Creating devicesim_devicesim2_1 ... 
Creating devicesim_devicesim3_1
Creating devicesim_devicesim1_1
Creating devicesim_devicesim2_1 ... done
Attaching to devicesim_devicesim3_1, devicesim_devicesim1_1, devicesim_devicesim2_1
devicesim1_1  | gNOI running on localhost:50001
devicesim3_1  | gNOI running on localhost:50003
devicesim3_1  | I0430 01:44:00.589325      12 gnoi_target.go:66] No credentials, setting Bootstrapping state.
devicesim3_1  | I0430 01:44:00.589809      12 gnoi_target.go:48] Starting gNOI server.
devicesim2_1  | gNOI running on localhost:50002
devicesim1_1  | I0430 01:44:01.702974      12 gnoi_target.go:66] No credentials, setting Bootstrapping state.
devicesim1_1  | I0430 01:44:01.703426      12 gnoi_target.go:48] Starting gNOI server.
devicesim2_1  | I0430 01:44:02.072945      12 gnoi_target.go:66] No credentials, setting Bootstrapping state.
devicesim2_1  | I0430 01:44:02.073391      12 gnoi_target.go:48] Starting gNOI server.
```

> This uses port mapping to make the devices available to gNOI clients and is the
> only option for running on Mac or Windows, and can be used with Linux too.

### Running on Linux
If you are fortunate enough to be using Docker on Linux, then you can use the
above method __or__ the alternative:

```bash
docker-compose -f docker-compose-linux.yml up
```

This will use the fixed IP addresses 172.25.0.11, 172.25.0.12, 172.25.0.13 for
device1-3. An entry must still be placed in your /etc/hosts file for all 3 like:
```bash
172.25.0.11 device1.opennetworking.org
172.25.0.12 device2.opennetworking.org
172.25.0.13 device3.opennetworking.org
```

> This uses a custom network 'simnet' in Docker and is only possible on Docker for Linux.
> If you are on Mac or Windows it is __not possible__ to route to User Defined networks,
> so the port mapping technique must be used.

> It is not possible to use the name mapping of the docker network from outside
> the cluster, so either the entries have to be placed in /etc/hosts or on some
> DNS server


## Run a single docker container
If you just want to run a single device, it is not necessary to run docker-compose.
It can be done just by docker directly, and can be handy for troubleshooting.
```bash
docker run --env "GNOI_TARGET=localhost" --env "GNOI_PORT=50001" -p "50001:50001" devicesim
```
To stop it use "docker kill"

## How to test the gNOI simulator? 

1. First you should  ssh into any of the targets using the following command:
```bash
sudo docker exec -it <Container_ID> /bin/bash
```

2. From the Docker, you can install  a Certificate and CA Bundle on a target that is in bootstrapping mode, accepting encrypted TLS connections using the following command:
```bash
gnoi_cert -target_addr localhost:50002  \
          -target_name "gnoi_localhost_50002" \ 
          -key certs/client1.key  \
          -ca certs/client1.crt  \
          -op provision  \
          -cert_id provision_cert \  
          -alsologtostderr
```

After executing the above command the output will be like the following:
```bash
devicesim2_1  | I0430 01:50:17.406240      12 server.go:59] Start Install request.
devicesim2_1  | I0430 01:50:17.420670      12 server.go:152] Success Install request.
devicesim2_1  | I0430 01:50:17.420732      12 manager.go:100] Notifying for: 1 Certificates and 1 CA Certificates.
devicesim2_1  | I0430 01:50:17.420754      12 gnoi_target.go:61] Found Credentials, setting Provisioned state.
devicesim2_1  | I0430 01:50:17.421202      12 gnoi_target.go:48] Starting gNOI server.
```

3. To get all installed certificate on a provisioned Target, you can run a command like the following:
```bash
gnoi_cert \
     -target_addr localhost:50002  \
     -target_name "gnoi_localhost_50002" \
     -key certs/client1.key  \
     -ca certs/client1.crt  \
     -op get  \
     -alsologtostderr
```

After the executing the above command, the output in the client side should be like: 
```bash
I0429 15:45:00.910217      84 gnoi_cert.go:226] GetCertificates:
{provision_cert: "gnoi_localhost_50002"}
```

and the the server should report a success message: 
```
devicesim2_1  | I0429 15:45:00.909347      10 server.go:292] Success GetCertificates.
```

