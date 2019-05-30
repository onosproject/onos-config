# Device Simulator

This is a docker VM that runs a gNMI and/or gNOI implementation 
supporting openconfig models.

Inspired by https://github.com/faucetsdn/gnmi 

All commands below assume you are in the __devicesim__ directory

## Simulator mode
The device simulator can operate in three modes, controlled
using **SIM_MODE** environment variable in the docker-compose file. 
1) SIM_MODE=1 as gNMI target only. The configuration is loaded by default from [target_configs/typical_ofsw_config.json](target_configs/typical_ofsw_config.json)
2) SIM_MODE=2 as gNOI target only. It supports *Certificate management* that can be used for certificate installation and rotation. 
3) SIM_MODE=3 both gNMI and gNOI targets simultaneously

## Run mode - localhost or network
Additionally the simulator can be run in
* localhost mode - use on Docker for Mac, Windows or Linux
* dedicated network mode - for use on Linux only 

> Docker for Mac or Windows does not support accessing docker images
> externally in dedicated network mode. Docker on Linux can run either.

## docker-compose
Docker compose manages the running of several docker images at once.

For example to run 3 **SIM_MODE=1** (gNMI only devices) and **localhost** mode, use: 
```bash
cd docker_compose
docker-compose -f docker-compose-gnmi.yml up
```

This gives an output like
```bash
Creating devicesim_devicesim3_1 ... 
Creating devicesim_devicesim1_1 ... 
Creating devicesim_devicesim2_1 ... 
Creating devicesim_devicesim3_1
Creating devicesim_devicesim2_1
Creating devicesim_devicesim1_1 ... done
Attaching to devicesim_devicesim3_1, devicesim_devicesim2_1, devicesim_devicesim1_1
devicesim3_1  | gNMI running on localhost:10163
devicesim2_1  | gNMI running on localhost:10162
devicesim1_1  | gNMI running on localhost:10161
```
> Use the -d mode with docker-compose to make it run as a daemon in the background


### Running on Linux
If you are fortunate enough to be using Docker on Linux, then you can use the
above method __or__ using the command below to start in **SIM_MODE=1** and **network** mode:

```bash
cd docker_compose
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
If you just want to run a single device, it is not necessary to run 
docker-compose. It can be done just by docker directly, and can be 
handy for troubleshooting. The following command shows how to run
a standalone simulator in SIM_MODE=3, localhost mode:
```bash
docker run --env "HOSTNAME=localhost" --env "SIM_MODE=3" \
    --env "GNMI_PORT=10164" --env "GNOI_PORT=50004" \
    -p "10164:10164" -p "50004:50004" onosproject/devicesim
```
To stop it use "docker kill"

## Create the docker image
By default the docker compose command will pull down the latest docker
image from the Docker Hub. If you need to build it locally, run:
```bash
docker build -t onosproject/devicesim:stable -f Dockerfile .
```

# Client tools for testing
You can access to the information about client tools for each SIM_MODE
including troubleshooting tips using the following links: 

[gNMI Client_User Manual](docs/gnmi_user_manual.md)

[gNOI Client_User Manual](docs/gnoi_user_manual.md)

