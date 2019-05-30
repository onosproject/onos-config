# gNOI Client Simulator User Manual

## How to test the gNOI simulator? 
1. First you should  ssh into any of the targets using the following command:
```bash
sudo docker exec -it <Container_ID> /bin/bash
```

2. From the Docker or your own machine, you can install  a certificate and CA bundle on a target that is in bootstrapping mode, accepting encrypted TLS connections using the following command:
```bash
gnoi_cert -target_addr localhost:50001  \
          -target_name "gnoi_localhost_50001" \
          -key certs/client1.key  \
          -ca certs/client1.crt  \
          -op provision  \
          -cert_id provision_cert \
          -alsologtostderr
```


After executing the above command the output will be like the following:
```bash
devicesim1_1  | I0430 01:50:17.406240      12 server.go:59] Start Install request.
devicesim1_1  | I0430 01:50:17.420670      12 server.go:152] Success Install request.
devicesim1_1  | I0430 01:50:17.420732      12 manager.go:100] Notifying for: 1 Certificates and 1 CA Certificates.
devicesim1_1  | I0430 01:50:17.420754      12 gnoi_target.go:61] Found Credentials, setting Provisioned state.
devicesim1_1  | I0430 01:50:17.421202      12 gnoi_target.go:48] Starting gNOI server.
```

3. To get all installed certificate on a provisioned Target, you can run a command like the following:
```bash
gnoi_cert \
     -target_addr localhost:50001  \
     -target_name "gnoi_localhost_50001" \
     -key certs/client1.key  \
     -ca certs/client1.crt  \
     -op get  \
     -alsologtostderr
```

After the executing the above command, the output in the client side should be like: 
```bash
I0429 15:45:00.910217      84 gnoi_cert.go:226] GetCertificates:
{provision_cert: "gnoi_localhost_50001"}
```

and the the server should report a success message: 
```
devicesim1_1  | I0429 15:45:00.909347      10 server.go:292] Success GetCertificates.
```

### How to install gNOI_cert on your machine? 
To install **gnoi_cert** on your own machine, you can use the following command: 
```bash
go get -u github.com/google/gnxi/gnoi_cert
go install -v github.com/google/gnxi/gnoi_cert
```

## Troubleshooting

### Connection Refused
If you get an error like
```bash
F0501 15:32:50.313449      30 gnoi_cert.go:220] Failed GetCertificates:rpc error: code = Unavailable desc = all SubConns are in TransientFailure, latest connection error: connection error: desc = "transport: Error while dialing dial tcp 127.0.0.1:50001: connect: connection refused"
```
That means the gNOI target is not running or you are provding wrong ip:port information to the gnoi_cert command. 


#### TCP diagnosis
> This is not a concern with port mapping method using localhost and is for 
> the Linux specific option only

Starting with TCP - see if you can ping the device
1. by IP address e.g. 17.18.0.2 - if not it might not be up or there's some
   other network problem
2. by short name e.g. device1 - if not maybe your /etc/hosts file is wrong or
   DNS domain search is not opennetworking.org
3. by long name e.g. device1.opennetowrking.org - if not maybe your /etc/hosts
   file is wrong

For the last 2 cases make sure that the IP address that is resolved matches what
was given at the startup of the simulator with docker.

#### HTTP Diagnosis
If TCP shows reachability then try with HTTPS - it's very important to remember
that for HTTPS the address at which you access the server **must** match exactly
the server name in the server key's Common Name (CN) like __localhost__ or
__device1.opennetworking.org__ (and not an IP address!)

Try using cURL to determine if there is a certificate problem
```
curl -v https://localhost:50001 --key certs/client1.key --cert certs/client1.crt --cacert certs/onfca.crt
```
This might give an error like
```bash
* Rebuilt URL to: https://localhost:50001/
*   Trying 172.18.0.3...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 50001 (#0)
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
* SSL: certificate subject name 'device3.opennetworking.org' does not match target host name 'localhost'
* stopped the pause stream!
* Closing connection 0
* TLSv1.2 (OUT), TLS alert, Client hello (1):
curl: (51) SSL: certificate subject name 'device3.opennetworking.org' does not match target host name 'localhost'
```

> In this case the device at __localhost__ has a certificate for
> device3.opennetworking.org. HTTPS does not accept this as a valid certificate
> as it indicates someone might be spoofing the server. This happens today in
> your browser if you access a site through HTTPS whose certificate CN does not
> match the URL - it is just a fact of life with HTTPS, and is not peculiar to gNMI.

When device names and certificates match, then curl will reply with a message like:
```bash
curl: (92) HTTP/2 stream 1 was not closed cleanly: INTERNAL_ERROR (err 2)
```

> This means the HTTPS handshake was __successful__, and it has failed at the
> gNOI level - not surprising since we did not send it any gNOI payload.
