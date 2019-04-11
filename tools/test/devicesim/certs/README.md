# ONF Certificates
This is a chain that generates certificates for the device simulator

The certificate authority is created once and should not need to be regenerated. If it is see instructions at the end of this file.
To see the details of it use
```
openssl x509 -in onfca.crt -text -noout
```

## Generate new cert
Use the generate_certs.sh script

Usage:	[server|client] [INDEX]"
	[-h | --help]"
Options:
	TYPE            The type of certificate: server -c for client"
	INDEX           The index for the certificate (number - default 1)"
	[-h | --help]   Print this help"

## Regenerating Certificate Authority
The should be no need to recreate the CA as it is valid until April 2029, but if you do here are the steps to take from a bash shell.

```
export SUBJ="/C=US/ST=CA/L=MenloPark/O=ONF/OU=Engineering/CN=ca.opennetworking.org"
openssl req -newkey rsa:2048 -nodes -keyout onfca.key -subj $SUBJ
openssl req -key onfca.key -new -out onfca.csr -subj $SUBJ
openssl x509 -signkey onfca.key -in onfca.csr -req -days 3650 -out onfca.crt
rm onfca.csr
```


