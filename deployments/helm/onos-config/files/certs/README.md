This folder contains self-signed certificates for use in testing. _DO NOT USE THESE
CERTIFICATES IN PRODUCTION!_

The certificates were generated with the following commands:

```bash
touch index.txt
echo '01' > serial.txt
openssl req -x509 -nodes -newkey rsa:2048 -config openssl.cnf -subj '/C=US/CN=ONF' -keyout tls.cakey -out tls.cacrt
openssl req -new -nodes -config openssl.cnf -subj "/C=US/CN=config.onosproject.org" -keyout tls.key -out tls.req
openssl ca -batch -config openssl.cnf -keyfile tls.cakey -cert tls.cacrt -out tls.crt -infiles tls.req
```

The `openssl.cnf` configuration is:

```
dir = .

[ ca ]
default_ca = CA_default

[ CA_default ]
serial = $dir/serial.txt
database = $dir/index.txt
new_certs_dir = $dir
certificate  = $dir/ssl.ca
private_key = $dir/private/ssl.cakey
default_days = 36500
default_md  = sha256
preserve = no
email_in_dn  = no
nameopt = default_ca
certopt = default_ca
policy = policy_match

[ policy_match ]
commonName = supplied
countryName = optional
stateOrProvinceName = optional
organizationName = optional
organizationalUnitName = optional
emailAddress = optional

[ req ]
default_bits = 2048
default_keyfile = priv.pem
default_md = sha256
distinguished_name = req_distinguished_name
req_extensions = v3_req
encyrpt_key = no

[ req_distinguished_name ]

[ v3_ca ]
basicConstraints = CA:TRUE
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer:always

[ v3_req ]
basicConstraints = CA:FALSE
subjectKeyIdentifier = hash
```
