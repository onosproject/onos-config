# Device simulator

This is a docker VM that runs gNMI implementation supporting openconfig models

Inspired by https://github.com/faucetsdn/gnmi 

Everything below assumes you are in the __devicesim__ directory

The configuration is loaded by default from [typical_ofsw_config.json](targetconfigs/typical_ofsw_config.json)

## Create the docker image
```bash
docker build -t devicesim -f Dockerfile .
```

## Run 3 devices with Docker Compose
Run 3 devices with 
```bash
docker-compose up
```

This gives an output like
```bash
Recreating devicesim_devicesim2_1 ...
Recreating devicesim_devicesim2_1
Recreating devicesim_devicesim3_1 ...
Recreating devicesim_devicesim1_1 ...
Recreating devicesim_devicesim1_1
Recreating devicesim_devicesim1_1 ... done
Attaching to devicesim_devicesim2_1, devicesim_devicesim3_1, devicesim_devicesim1_1
devicesim3_1  | Please add '172.18.0.3 device3 device3.opennetworking.org' to /etc/hosts and access with gNMI client at device3:10161
devicesim2_1  | Please add '172.18.0.4 device2 device2.opennetworking.org' to /etc/hosts and access with gNMI client at device2:10161
devicesim1_1  | Please add '172.18.0.2 device1 device1.opennetworking.org' to /etc/hosts and access with gNMI client at device1:10161
```

___Note___
> The trouble with the default network is that IP addresses cannot be assigned and
> so each time Docker starts up the assignment of IP addresses to hostnames will vary.

> The prompt given above give entries that __must__ be added to your local /etc/hosts
> file. With Docker it is __not possible__ to access the name resolution scheme of
> the default network, and so entries must be added manually. An alternative is to
> run your own local DNS server on your PC.

### Running on Linux
If you are fortunate enough to be using Docker on Linux, then you can use the alternative:
```bash
docker-compose -f docker-compose-linux.yml up
```
This will use the fixed IP addresses 172.25.0.11, 172.25.0.12, 172.25.0.13 for
device1-3. An entry must still be placed in your /etc/hosts file for all 3, but
at least they will not change each time you restart the cluster.
> This uses a custom network 'simnet' in Docker and is only possible on Docker for Linux.
> If you are on Mac or Windows it is __not possible__ to route to User Defined networks,
> so the __default__ Docker network must be used.


## Get the capabilities
```bash
gnmi_cli -address device1.opennetworking.org:10161 \
       -capabilities \
       -timeout 5s \
       -client_crt certs/client1.crt \
       -client_key certs/client1.key \
       -ca_crt certs/onfca.crt \
       -alsologtostderr
```

If you get
```bash
E0416 15:23:08.099600   22997 gnmi_cli.go:180] could not create a gNMI client: Dialer(device1.opennetworking.org:10161, 5s): context deadline exceeded
```
Please see the [troubleshooting](#deadline-exceeded) section below.

## Retrieve the hostname
```bash
gnmi_cli -address device1.opennetworking.org:10161 \
       -get \
       -proto "path: <elem: <name: 'system'> elem:<name:'config'> elem: <name: 'hostname'>>" \
       -timeout 5s \
       -client_crt certs/client1.crt \
       -client_key certs/client1.key \
       -ca_crt certs/onfca.crt \
       -alsologtostderr
```

This gives a response like
```bash
notification: <
  timestamp: 1555427003418090847
  update: <
    path: <
      elem: <
        name: "system"
      >
      elem: <
        name: "config"
      >
      elem: <
        name: "hostname"
      >
    >
    val: <
      string_val: "device1"
    >
  >
>
```

## Troubleshooting
### Run a single docker container
If you just want to run a single device, it is not necessary to run docker-compose.
It can be done just by docker directly, and can be handy for troubleshooting.
```bash
docker run devicesim
```
To stop it user "docker kill"


### Deadline exceeded
If you get an error like
```bash
E0416 15:23:08.099600   22997 gnmi_cli.go:180] could not create a gNMI client:
Dialer(device1.opennetworking.org:10161, 5s): context deadline exceeded
```

or anything about __deadline exceeded__, then it is **always** related to the
transport mechanism above gNMI i.e. TCP or HTTPS

#### TCP diagnosis
Starting with TCP - see if you can ping the device
1. by IP address e.g. 17.18.0.2 - if not it might not be up or there's some
   other network problem
2. by short name e.g. device1 - if not maybe your /etc/hosts file is wrong

For both make sure that the IP address that is resolved matches what was given
at the startup of the simulator.

#### HTTP Diagnosis
If TCP shows reachability then try with HTTPS - it's very important to remember
that for HTTPS the address at which you access the server **must** match exactly
the server name in the server key's Common Name (CN) like
__device1.opennetworking.org__ (and not an IP address!)

Try using cURL to determine if there is a certificate problem
```
curl -v https://device2.opennetworking.org:10161 --key certs/client1.key --cert certs/client1.crt --cacert certs/onfca.crt
```
This might give an error like
```bash
* Rebuilt URL to: https://device2.opennetworking.org:10161/
*   Trying 172.18.0.3...
* TCP_NODELAY set
* Connected to device2.opennetworking.org (172.18.0.3) port 10161 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* successfully set certificate verify locations:
*   CAfile: certs/onfca.crt
  CApath: /etc/ssl/certs
* TLSv1.2 (OUT), TLS handshake, Client hello (1):
* TLSv1.2 (IN), TLS handshake, Server hello (2):
* TLSv1.2 (IN), TLS handshake, Certificate (11):
* TLSv1.2 (IN), TLS handshake, Server key exchange (12):
* TLSv1.2 (IN), TLS handshake, Request CERT (13):
* TLSv1.2 (IN), TLS handshake, Server finished (14):
* TLSv1.2 (OUT), TLS handshake, Certificate (11):
* TLSv1.2 (OUT), TLS handshake, Client key exchange (16):
* TLSv1.2 (OUT), TLS handshake, CERT verify (15):
* TLSv1.2 (OUT), TLS change cipher, Client hello (1):
* TLSv1.2 (OUT), TLS handshake, Finished (20):
* TLSv1.2 (IN), TLS handshake, Finished (20):
* SSL connection using TLSv1.2 / ECDHE-RSA-AES256-GCM-SHA384
* ALPN, server accepted to use h2
* Server certificate:
*  subject: C=US; ST=CA; L=MenloPark; O=ONF; OU=Engineering; CN=device3.opennetworking.org
*  start date: Apr 16 14:40:46 2019 GMT
*  expire date: Apr 15 14:40:46 2020 GMT
* SSL: certificate subject name 'device3.opennetworking.org' does not match target host name 'device2.opennetworking.org'
* stopped the pause stream!
* Closing connection 0
* TLSv1.2 (OUT), TLS alert, Client hello (1):
curl: (51) SSL: certificate subject name 'device3.opennetworking.org' does not match target host name 'device2.opennetworking.org'
```

In this case the name __device2.opennetworking.org__ in /etc/hosts was incorrectly
pointed at the IP address for __device3__. HTTPS does not accept this as a valid
certificate as it indicates someone might be spoofing the server. This happens
today in your browser if you access a site through HTTPS whose certificate CN does
not match the URL - it is just a fact of life with HTTPS, and so with gNMI.

When device names and certificates match then curl will reply with a message like
```bash
curl: (92) HTTP/2 stream 1 was not closed cleanly: INTERNAL_ERROR (err 2)
```

This actually means the HTTPS handshake was __successful__, and it has failed at the
gNMI level - not surprising since we did not send it any payload. At this stage
you should be able to use **gnmi_cli**.


