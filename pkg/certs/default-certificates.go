// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package certs

/*
All of these are copied over from onos-config/tools/test/devicesim/certs
*/


/*
OnfCaCrt is the default CA certificate
Certificate:
    Data:
        Version: 1 (0x0)
        Serial Number:
            de:f7:d7:d2:37:da:b1:49
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = ca.opennetworking.org
        Validity
            Not Before: Apr 11 09:06:13 2019 GMT
            Not After : Apr  8 09:06:13 2029 GMT
        Subject: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = ca.opennetworking.org
*/
const OnfCaCrt = `
-----BEGIN CERTIFICATE-----
MIIDYDCCAkgCCQDe99fSN9qxSTANBgkqhkiG9w0BAQsFADByMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCU1lbmxvUGFyazEMMAoGA1UECgwDT05G
MRQwEgYDVQQLDAtFbmdpbmVlcmluZzEeMBwGA1UEAwwVY2Eub3Blbm5ldHdvcmtp
bmcub3JnMB4XDTE5MDQxMTA5MDYxM1oXDTI5MDQwODA5MDYxM1owcjELMAkGA1UE
BhMCVVMxCzAJBgNVBAgMAkNBMRIwEAYDVQQHDAlNZW5sb1BhcmsxDDAKBgNVBAoM
A09ORjEUMBIGA1UECwwLRW5naW5lZXJpbmcxHjAcBgNVBAMMFWNhLm9wZW5uZXR3
b3JraW5nLm9yZzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMEg7CZR
X8Y+syKHaQCh6mNIL1D065trwX8RnuKM2kBwSu034zefQAPloWugSoJgJnf5fe0j
nUD8gN3Sm8XRhCkvf67pzfabgw4n8eJmHScyL/ugyExB6Kahwzn37bt3oT3gSqhr
6PUznWJ8fvfVuCHZZkv/HPRp4eyAcGzbJ4TuB0go4s6VE0WU5OCxCSlAiK3lvpVr
3DOLdYLVoCa5q8Ctl3wXDrfTLw5/Bpfrg9fF9ED2/YKIdV8KZ2ki/gwEOQqWcKp8
0LkTlfOWsdGjp4opPuPT7njMBGXMJzJ8/J1e1aJvIsoB7n8XrfvkNiWL5U3fM4N7
UZN9jfcl7ULmm7cCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEAIh6FjkQuTfXddmZY
FYpoTen/VD5iu2Xxc1TexwmKeH+YtaKp1Zk8PTgbCtMEwEiyslfeHTMtODfnpUIk
DwvtB4W0PAnreRsqh9MBzdU6YZmzGyZ92vSUB3yukkHaYzyjeKM0AwgVl9yRNEZw
Y/OM070hJXXzJh3eJpLl9dlUbMKzaoAh2bZx6y3ZJIZFs/zrpGfg4lvBAvfO/59i
mxJ9bQBSN3U2Hwp6ioOQzP0LpllfXtx9N5LanWpB0cu/HN9vAgtp3kRTBZD0M1XI
Ctit8bXV7Mz+1iGqoyUhfCYcCSjuWTgAxzir+hrdn7uO67Hv4ndCoSj4SQaGka3W
eEfVeA==
-----END CERTIFICATE-----
`

/*
DefaultClientKey is the default client key
openssl rsa -in client1.key -text -noout
Private-Key: (2048 bit)
*/
const DefaultClientKey = `
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDmZHXagZc/64MP
sPNBl1OD3p4dZNzRQo/CJ7YjksstWxGgfY87mzYHmvqv8Tw5QbjSkZvq2QNEgnx4
LRn/o28GTOfh+wnmDRRbqxWVd9p/TL0mcxK9XTn6D77vlrv4JHHUEbcY2MZZuR7W
bHx3pKtDwPWyMjcvQP7faaLRZT2M7ZTD1he9eDFCYv0qvF2vcz9pk72xMlm0hPQQ
RjeoGk+/51oBu4/mb66GITNhk/ZsvLDLMbJ+tSaP1+Mb8PWS4wyFoQhQlDZVNwgC
kob8pvcyTdvdUJ9O+kQcxWsoyjI0dBdeL7pfXTr5y7Za2g+/5TlUVWi5+NFZJtw6
ZDuogcGLAgMBAAECggEBAIc9VUjsZSJqVsaxMjnAYI+578qFWHGlxslLkkkTdByt
po005w0wMOkJ+jmpO5bIk3tXadTTim1+wx2wK+C5yQRDxKIMQGVALEEbDlJsxl+P
ZkDZr5hkzxGQiJ4PN0uT6RV5SKdXKCem2Qk5KV751GazMAZoH6inWHVAhwiviw/b
kSJmXcQifxB9R5Br+yCdkRNGg+EtadxAkRtZdW0N0H6LwWxsl32I4o1WM3N2Tyag
kpKPPZ5J5U+279Rpz7W4JAbGzWBOL0Wc2pz5p+aKVTWia0MoqzHR4P4YnkGM+w9Y
j6+Nemdedx62KPhOnQH1uvuG3vnOtt2Ss5OLxePgmjECgYEA9bVguF1D5rpp6MSK
2izZt0mNqhiozm84W2UrAwDhtW5tptW2JBPj2T05+PbEOUEgsvucWfmhZoBXNRCw
IlLQZh46LJFXyW1Awn3PuYquruF61phDoqU9Ou5skJrh0ez+vX872HkH4KW3MfWq
w3LW4qXt6z+lBgPY8hNAlis3WE0CgYEA8Ara5J915ZoVll1As84H61NHmkyMFENh
PjUJqL6tPxvZ+lkBeA157o6mrIgNmG5bLnzonpT4rqemewxEYL39sJ6CVzHRFy8I
F0VNLzZbYizrPLRvT+Gkh0jf6W7Iarzmcdb8cMDxQ+9LmwR/Q3XAD8ntqzrbwVl5
FOZlGq2ZbTcCgYEAuMULlbi07hXyvNLH4+dkVXufZ3EhyBNFGx2J6blJAkmndZUy
YhD+/4cWSE0xJCkAsPebDOI26EDM05/YBAeopZJHhupJTLS2xUsc4VcTo3j2Cdf4
zJ9b2yweQePmuxlwOwop89CYBuw3Rf+KyW1bgJbswkJbE5njE688m3CmLuUCgYAf
K2mtEj++5rky4z0JnBFPL2s20AXIg89WwpBUhx37+ePeLDySmD1jCsb91FTfnETe
zn1uSi3YkBCAHeGrJkCQ9KQ8Kk3aUtMcInWZUdef8fFB2rQxjT1OC9p3d1ky8wCB
e8cf5Q3vIl2Q7Y6Q9fNQmYnxGB19B98/JYOvaSdpFQKBgFBJ+tdJ5ghXSdvAzGno
trQlL1AYW/kYsxZaALd1R+vK3vxeHOtUWiq3923QttYsVXPRQe1TEEdxlOb7+hwE
g5NVOIsDpB1OqjQRb9PjipANkHQRKgrYFB20ZQUoaOMckhlVyqE6WcanGpUxJ0xg
1F0itWrqPGEs83BRQI/aLlsj
-----END PRIVATE KEY-----
`

/*
DefaultClientCrt is the default client certificate
    Data:
        Version: 1 (0x0)
        Serial Number:
            e5:ec:d1:7a:7a:47:df:71
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = ca.opennetworking.org
        Validity
            Not Before: Apr 11 11:16:23 2019 GMT
            Not After : Apr 10 11:16:23 2020 GMT
        Subject: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = client1.opennetworking.org
*/
const DefaultClientCrt = `
-----BEGIN CERTIFICATE-----
MIIDZTCCAk0CCQDl7NF6ekffcTANBgkqhkiG9w0BAQsFADByMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCU1lbmxvUGFyazEMMAoGA1UECgwDT05G
MRQwEgYDVQQLDAtFbmdpbmVlcmluZzEeMBwGA1UEAwwVY2Eub3Blbm5ldHdvcmtp
bmcub3JnMB4XDTE5MDQxMTExMTYyM1oXDTIwMDQxMDExMTYyM1owdzELMAkGA1UE
BhMCVVMxCzAJBgNVBAgMAkNBMRIwEAYDVQQHDAlNZW5sb1BhcmsxDDAKBgNVBAoM
A09ORjEUMBIGA1UECwwLRW5naW5lZXJpbmcxIzAhBgNVBAMMGmNsaWVudDEub3Bl
bm5ldHdvcmtpbmcub3JnMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
5mR12oGXP+uDD7DzQZdTg96eHWTc0UKPwie2I5LLLVsRoH2PO5s2B5r6r/E8OUG4
0pGb6tkDRIJ8eC0Z/6NvBkzn4fsJ5g0UW6sVlXfaf0y9JnMSvV05+g++75a7+CRx
1BG3GNjGWbke1mx8d6SrQ8D1sjI3L0D+32mi0WU9jO2Uw9YXvXgxQmL9Krxdr3M/
aZO9sTJZtIT0EEY3qBpPv+daAbuP5m+uhiEzYZP2bLywyzGyfrUmj9fjG/D1kuMM
haEIUJQ2VTcIApKG/Kb3Mk3b3VCfTvpEHMVrKMoyNHQXXi+6X106+cu2WtoPv+U5
VFVoufjRWSbcOmQ7qIHBiwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBRBR6LTFEU
SWeEeguMsbHxN/6NIZuPejib1q9fTHeZ9cnIHIOLJaZzHiMZn5uw8s6D26kveNps
iCr4O8xOjUa0uwbhMTgm3wkODLlV1DwGjFWk8v5UKGWqUQ94wVMQ16YMIR5DgJJM
0DUzVcoFz+vLnMrDZ0AEk5vra1Z5KweSRvwHX7dJ6FIW7X3IgqXTqJtlV/D/vIi3
UfBnjzqOy2LVfBD7du7i5NbTHfTUpoTvddVwQaKCuQGYHocoQvQD3VQcQDh1u0DD
n2GkeEDLaDAGFAIO+PDg2iT8BhKeEepqswid9gYAhZcOjrlnl6smZo7jEzBj1a9Q
e3q1STjfQqe8
-----END CERTIFICATE-----
`
