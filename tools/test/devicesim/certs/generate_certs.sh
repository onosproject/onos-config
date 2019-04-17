#!/bin/sh

SUBJBASE="/C=US/ST=CA/L=MenloPark/O=ONF/OU=Engineering/"
DEVICE=${1:-device1.opennetworking.org}
SUBJ=${SUBJBASE}"CN="${DEVICE}

print_usage() {
    echo "Generate a certificate."
    echo
    echo "Usage:     <devicename>"
    echo "           [-h | --help]"
    echo "Options:"
    echo "    DEVICENAME      e.g. device1.opennetworking.org or localhost"
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

rm -f ${DEVICE}.*

# Generate Server Private Key
openssl req \
        -newkey rsa:2048 \
        -nodes \
        -keyout ${DEVICE}.key \
	-noout \
        -subj $SUBJ \
	 > /dev/null 2>&1

# Generate Req
openssl req \
        -key ${DEVICE}.key \
        -new -out ${DEVICE}.csr \
        -subj $SUBJ \
	 > /dev/null 2>&1

# Generate x509 with signed CA
openssl x509 \
        -req \
        -in ${DEVICE}.csr \
        -CA onfca.crt \
        -CAkey onfca.key \
        -CAcreateserial \
	-days 365 \
        -out ${DEVICE}.crt \
	 > /dev/null 2>&1

rm ${DEVICE}.csr

echo " == Certificate Generated: "${DEVICE}.crt" =="
openssl verify -verbose -purpose sslserver -CAfile onfca.crt ${DEVICE}.crt > /dev/null 2>&1
exit $?
#To see full details run 'openssl x509 -in "${TYPE}${INDEX}".crt -text -noout'




