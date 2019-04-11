#Device simulator

This is a docker VM that runs gNMI implementation supporting openconfig models

Inspired by https://github.com/faucetsdn/gnmi 

## Create the docker image
docker build -t devicesim -f Dockerfile .


## Run 3 devices with Docker Compose
Run 3 devices with 
```
docker-compose up
```

To access these by gNMI from your desktop machine their ip addresses must be added to your machines
__/etc/hosts__ file. See here 
```
172.25.0.11 device1.opennetworking.org
172.25.0.12 device2.opennetworking.org
172.25.0.13 device3.opennetworking.org
```

## Get the capabilities
```
gnmi_cli -address device1.opennetworking.org:10161 \
       -capabilities \
       -timeout 5s \
       -client_crt certs/client1.crt \
       -client_key certs/client1.key \
       -ca_crt certs/onfca.crt \
       -alsologtostderr
```

## Retrieve the hostname
```
gnmi_cli -address device1.opennetworking.org:10161 \
       -get \
       -proto "path: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>" \
       -timeout 5s \
       -client_crt certs/client1.crt \
       -client_key certs/client1.key \
       -ca_crt certs/onfca.crt \
       -alsologtostderr
```

## Run a single docker container
```
docker run devicesim
```



