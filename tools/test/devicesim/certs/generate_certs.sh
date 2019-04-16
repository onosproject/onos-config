#!/bin/sh

SUBJBASE="/C=US/ST=CA/L=MenloPark/O=ONF/OU=Engineering/"
TYPE=${1:-device}
INDEX=${2:-1}
SUBJ=${SUBJBASE}"CN="${TYPE}${INDEX}".opennetworking.org"

print_usage() {
    echo "Generate a certificate."
    echo
    echo "Usage:     <device|client> [INDEX]"
    echo "           [-h | --help]"
    echo "Options:"
    echo "    TYPE            The type of certificate: device -c for client"
    echo "    INDEX           The index for the certificate (number - default 1)"
    echo "    [-h | --help]   Print this help"
    echo "";
}

# Print usage
if [ "${1}" = "-h" -o "${1}" = "--help" ]; then
    print_usage
    exit 0
fi

if [ "${PWD##*/}" != "certs" ]; then
    cd certs
fi

rm -f ${TYPE}${INDEX}.*

# Generate Server Private Key
openssl req \
        -newkey rsa:2048 \
        -nodes \
        -keyout ${TYPE}${INDEX}.key \
	-noout \
        -subj $SUBJ \
	 > /dev/null 2>&1

# Generate Req
openssl req \
        -key ${TYPE}${INDEX}.key \
        -new -out ${TYPE}${INDEX}.csr \
        -subj $SUBJ \
	 > /dev/null 2>&1

# Generate x509 with signed CA
openssl x509 \
        -req \
        -in ${TYPE}${INDEX}.csr \
        -CA onfca.crt \
        -CAkey onfca.key \
        -CAcreateserial \
	-days 365 \
        -out ${TYPE}${INDEX}.crt \
	 > /dev/null 2>&1

rm ${TYPE}${INDEX}.csr

echo " == Certificate Generated: "${TYPE}${INDEX}.crt" =="
openssl verify -verbose -purpose sslserver -CAfile onfca.crt ${TYPE}${INDEX}.crt > /dev/null 2>&1
exit $?
#To see full details run 'openssl x509 -in "${TYPE}${INDEX}".crt -text -noout'




