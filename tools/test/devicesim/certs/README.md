# ONF Certificates
This is a chain that generates certificates for the device simulator

```
    Certificate Authority (ca.opennetworking.org)
     /                       \     \---- localhost
    /                         \
client1.opennetworking.org     \---- <hostnames given as argument>
```

The certificate authority (CA) and the client1 and localhosts certificates are
created permanently, versioned in git and should not need to be regenerated.

To see the details of each use:
```
openssl x509 -in onfca.crt -text -noout
openssl x509 -in client1.crt -text -noout
openssl x509 -in localhost.crt -text -noout
```

> Device specific certificates can be created when the device simulator starts up
> when running in Linux ([see devicesim](../README.md)) and are keyed to the 
> GNMI_TARGET evnironment variable passed in to the simulator.

## Generate new cert
Use the generate_certs.sh script
```bash
Usage:	<devicename>
	[-h | --help]
Options:
	DEVICENAME      e.g. device1.opennetworking.org or localhost
	[-h | --help]   Print this help"
```

## Regenerating Certificate Authority
The should be no need to recreate the CA as it is valid until April 2029, but
if you do here are the steps to take from a bash shell.

```
export SUBJ="/C=US/ST=CA/L=MenloPark/O=ONF/OU=Engineering/CN=ca.opennetworking.org"
openssl req -newkey rsa:2048 -nodes -keyout onfca.key -subj $SUBJ
openssl req -key onfca.key -new -out onfca.csr -subj $SUBJ
openssl x509 -signkey onfca.key -in onfca.csr -req -days 3650 -out onfca.crt
rm onfca.csr
```

## Signed Certificate authority
At some stage in this project it is expected that a Certificate Authority will
be created from some well know chain e.g. GlobalSign or
[Comodo](https://comodosslstore.com/promoads/cheap-comodo-ssl-certificates.aspx)

This would mean that any certificates derived from this chain would be well
known in the major operating systems and then utilities like cURL would not have
to import the CA every time.

