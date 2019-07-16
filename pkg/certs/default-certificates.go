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

/*
Package certs is set of default certificates serialized in to string format.
*/
package certs

// Default Client Certificate name
const (
	Client1Key = "client1.key"
	Client1Crt = "client1.crt"
)

/*
All of these are copied over from https://github.com/onosproject/simulators/tree/master/pkg/certs
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

/*
DefaultLocalhostKey is the default localhost server key
openssl rsa -in client1.key -text -noout
Private-Key: (2048 bit)
*/
const DefaultLocalhostKey = `
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC4zBmctGU/0LQ4
uJAtny0Zh/NwAVAgjAlVE9Y3hh1fTiXB7eAPhLrCa3KCzRlQAUCvorNPKYhlHV70
CL8ACWVrstxeY635I3UGanPZhdNRv1h6hW8hympqIKO1otWDV6pwi0gXo6Zco8QQ
/3LriBlsGn34kvP52Sh4n/id38CCw34OgkItfZb8VeSkXdNQHN1sfB/z0fJXGdIz
Io5JGGfgiw/i5ccesyXxfN8A3vBhKE1sUJzeHZGEzkdmo/ZbfpVQt4H6aKurYfwD
xDQjVasf+Lq+WDKd2OzN+BZ+YBMNWUcmZkAh8zZmQWhCttLL484GOeCtqq5HGsTD
JvfkHSAvAgMBAAECggEBAIi5ORnfvimA2FY+9y1J36xMEaiE0CvEcAMqMgvShljF
ENpyjJvur964cHimFlxDEQDhd5jSOb/WAzK6ZdY5HXiZVMHhLg5uVV7x09TUVozc
7TF5F8gAYssyau0wFJige9HYuvYCdkuEPsP0u6nXgDejQiBvWWM5b+APO3pS2bPk
fbyXPnp+xUQAQhH/m+VCCG4hlC1bKulkY4yWKp5wrKtj2Xun0Vh4Iu9UJiH/EuQ6
mwpUeWLECO5OYDKQWH90iIVDZufe+8yw1VZrN82cTWL86jetgPEXz6klWGUu1gHU
r57xz0Nb4rhFPs49aw9Yj/LswF1I5zdF6EiT0H5aRoECgYEA2tNBEo4ta3JW4wF/
/3AKAE79RV+06z74x3Id5w2ED8TRq3SRu5oMmR9kJhe6owNg8WcbK/h/LuI5+CPD
V8Uo8HGQs5VisSJGJdCul8gA2bDGPEokGezdEdE6UUaNxpgRU484owZau80c4Q/Y
B4evLdJL9KIaSB5oPHGfu65i41kCgYEA2DD29X6i1GOSwzKXTXNdrZjxxOqeZiVv
/+TTiRffIUNMObtR6B8wWi5Y+oPUUj+Xop1vM7L9ZkwfkuDMtaZbryA37rsoAKP9
Sdlemyt0cB+cL7MN04Od9UD4YzbapRGAoFJduzzneQN0PBUc6wvB6cMBH2UZsEjQ
GdV0r/iC1scCgYEA0pg9J/5s99syg4YOCWdqOKHMXded5kjUZB4PaS44ynRA1SF6
n3HCbhsn5wEvPXMi+TChlc+xlw1hfM3uUaoNnFmvSSWbtZ2mpP4RCUISj27xWVSB
KfIrT9pspYuhJl9zTVeoyjxzVgowoOj+n0CV9yNMtkLLyFx7NLClaZqK0QECgYAj
rECz1YeMwDlxWCG7N/QXNwt90LD+beMDOIDnODcrR+2GATDMuojB+K/Z9nLMd43P
2WaGA1zoylrTY6CjwKWUSh6wl9VL9cNPsjx4Ij1+WtjszgDUC/2+gE/8HwsI/dBZ
o/2vbadMQpOlbl5tMm124ySGR6prejhMavpsJvd/9QKBgGBN5jGWqdezvFx1dY0S
uZAlqQ3h5w/0MMGSmaM+yN30wjFdWd3yMAxNgIp4Oj1noqvuqXiXnBjN6YER3GNx
hpjimxkZY9ogyVGz+RP9YqGKKbUtBL/8zZ/LYbWGvo9yJ6HxwO397EhyXNhCvwyF
sDhPaK+DnzwfkjBk3kXNve4o
-----END PRIVATE KEY-----
`

/*
DefaultLocalhostCrt is the default localhost server certificate
Certificate:
    Data:
        Version: 1 (0x0)
        Serial Number:
            a7:0e:38:d6:1c:87:53:46
    Signature Algorithm: sha256WithRSAEncryption
        Issuer: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = ca.opennetworking.org
        Validity
            Not Before: Apr 16 17:35:40 2019 GMT
            Not After : Apr 15 17:35:40 2020 GMT
        Subject: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = localhost
*/
const DefaultLocalhostCrt = `
-----BEGIN CERTIFICATE-----
MIIDVDCCAjwCCQCnDjjWHIdTRjANBgkqhkiG9w0BAQsFADByMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCU1lbmxvUGFyazEMMAoGA1UECgwDT05G
MRQwEgYDVQQLDAtFbmdpbmVlcmluZzEeMBwGA1UEAwwVY2Eub3Blbm5ldHdvcmtp
bmcub3JnMB4XDTE5MDQxNjE3MzU0MFoXDTIwMDQxNTE3MzU0MFowZjELMAkGA1UE
BhMCVVMxCzAJBgNVBAgMAkNBMRIwEAYDVQQHDAlNZW5sb1BhcmsxDDAKBgNVBAoM
A09ORjEUMBIGA1UECwwLRW5naW5lZXJpbmcxEjAQBgNVBAMMCWxvY2FsaG9zdDCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALjMGZy0ZT/QtDi4kC2fLRmH
83ABUCCMCVUT1jeGHV9OJcHt4A+EusJrcoLNGVABQK+is08piGUdXvQIvwAJZWuy
3F5jrfkjdQZqc9mF01G/WHqFbyHKamogo7Wi1YNXqnCLSBejplyjxBD/cuuIGWwa
ffiS8/nZKHif+J3fwILDfg6CQi19lvxV5KRd01Ac3Wx8H/PR8lcZ0jMijkkYZ+CL
D+Llxx6zJfF83wDe8GEoTWxQnN4dkYTOR2aj9lt+lVC3gfpoq6th/APENCNVqx/4
ur5YMp3Y7M34Fn5gEw1ZRyZmQCHzNmZBaEK20svjzgY54K2qrkcaxMMm9+QdIC8C
AwEAATANBgkqhkiG9w0BAQsFAAOCAQEAomXDe4bZl0JhL3FGeYny34qLZcsEjD+l
Y79a4iNIhP97mPNbC9k0E1K9zjb8Kkl5/Gvcfr+IIMXLfuAxP1HI55YdAWKyMUsY
zVujpMY29bcZJIdO/inu1A00F3CzQmyxOl6wXF3unbPZZe4QWIgtsWDGFKmpcV8Z
3O+JwUFbXcAENFGYp+dL1MhyOClXq5Hj0hQNoRTOBdQ9SzPQomcce9L/VhUtSjBz
ZCPSExNq6PUzUjocUGdaDI2Wjjt/IY6rd5AYnjkie28wt64r1EO1MGZiHiXx/NNO
/utYBBcAzIsc63hM3Cjo2h0uFti3Sdza1Czq1t/NlOL9/2qtxVdyyg==
-----END CERTIFICATE-----
`

/*
DefaultOnosConfigKey is the default onos-config server key
openssl rsa -in onso-config.key -text -noout
Private-Key: (2048 bit)
*/
const DefaultOnosConfigKey = `
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCZn2yJTuwzOI08
3/2/OaIwzQzvzuGBnp4ttZGxrSi0KdX7r19Vu2dI5eEDbSNNYa4+9xSwT0lTf9u0
/9i6mJYgejLNJU0CnAZDGxTLuxMHeGjPjeSpdVGuXq4qOj+kXKmdpATXZmyXwZlK
gamh7/u66Wcrgn2y6eK1iCIvK+L0U+vwAa6NokUJwZlFtLBPmE0wwZxnW9LMyWkz
K0KANCsua3uVBSFclREsnM+jw4V2Gk9smXxfRV1HWkP7xsZuGi8NHq8A1/JC1gE5
CNLVsnD/ePHV3vHYFzVTu66Q9hDJLGpyu2cTpjUv44p7/3fQsaz3fYuT31nAj5//
l9ZHJwXRAgMBAAECggEAVEwRCL+QCQNNLUxUNyxu/YxnPugtAi2B6t8pVXAJV+Nl
EjjHfYnaQTwzXufyaTHipZZ7ecvoFrOgYg/KY4n7R1MGsV94hKgNH6Gqpai/5meC
S/I2uW4xJhe6Rl20MoLOaDxqk7AWgqevcBz6cmv3nDcbb9qpExYYWziaWXwhi6Pv
unvmZGzpAu9ehb4BWlQmHVqIDAcLijTKD7uxHl9yiM7zG5KLzpPCmr4drWjyrsE+
oo7fA0GhLpjQCJ6Ye+2hBYNvMJLmvcgex++16vQDszhIy9rTtlFCqbPaADF/9bOn
rjTMPPYWijMFx0lltHhPq09527iL/Yt6szg/PbNOyQKBgQDIZjdNfNXs128Qm7Zq
opo6rQU3j17/QINUBOx/+xCWnNohBbo+pGHP6Z8Ld4Lx6P1LPoWUw5tX4v8fwXgJ
Rn3gnwTAiXx/S/dHEkiDQLMU7nlpkwRNUA00DDqAz8dZIgRZeHAE1PRxBihCjXVf
Z4PKTHsbomxOrVn+NuAV80qr2wKBgQDEPs47MTUI96Xh8BuixVAwi2QVo01QB5Dl
x/ds1v/IDHen9sV1Yjl7gLC9YpO/7kweFDZRt6Zm1IRh4Qy9yqtDUSAQEOtoI6A1
FvHCBC06NzYISBFNdmRnkS3AJEb9piJY62YeP9+vWJqMuVc39igIbYoGR654uszJ
gEH2lgu6wwKBgQDHfCjU882oBBRFPhvqLo7ElfM5iXiRMtEIVBZwl6W9p8njUWZC
cTQE2ZQ+v+sTkFCEFGq42bbLV+WK4PXylb88WE9Msg/CUAaJMwQH0+Hwlis6EuUX
aPabtwiNrUfNzHTz81XfGXVzBSQSi+oo3Exulo99xMN31kxdKJcMgrD0PQKBgEx+
1tDH645lSin5+CvIket6SjcNArPxXw/SlKW+YNHP2kyEqo+JDDMSBNKtvD4SW2VW
J55O4fQvXrLwkJDikUOaOc9JaRmc2XQYT4B7NE3++3ba8LOrNJQSSS0edvWkbrsO
dy3PZBfrh8LW9CKCNzShzi2If3/cALuC3TOLZWMVAoGAOHexq8+7EoO9FXcqyRfN
tgv108T3UD7LNcH2Fj5sx+o/NN/OcGcCCb40I5Jrh305a+GPRG1RKW7fmD8UaMaM
L9Q8NQ7nUfUvl0RR9vwAgcvLy1zD+IgqAzqF0MN/ejIHTACIgL4S0cXlUyCk5gQn
wXqkSZWW53eg8HKb/cVMLoM=
-----END PRIVATE KEY-----
`

/*
DefaultOnosConfigCrt is the default onos-config server certificate
Certificate:
    Data:
        Version: 1 (0x0)
        Serial Number:
            56:56:04:5e:9b:45:15:87:d7:24:3d:2a:22:21:df:87:11:e0:f2:0b
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = ca.opennetworking.org
        Validity
            Not Before: Jul 16 18:38:15 2019 GMT
            Not After : Jul 13 18:38:15 2029 GMT
        Subject: C = US, ST = CA, L = MenloPark, O = ONF, OU = Engineering, CN = onos-config
*/
const DefaultOnosConfigCrt = `
-----BEGIN CERTIFICATE-----
MIIDYTCCAkkCFFZWBF6bRRWH1yQ9KiIh34cR4PILMA0GCSqGSIb3DQEBCwUAMHIx
CzAJBgNVBAYTAlVTMQswCQYDVQQIDAJDQTESMBAGA1UEBwwJTWVubG9QYXJrMQww
CgYDVQQKDANPTkYxFDASBgNVBAsMC0VuZ2luZWVyaW5nMR4wHAYDVQQDDBVjYS5v
cGVubmV0d29ya2luZy5vcmcwHhcNMTkwNzE2MTgzODE1WhcNMjkwNzEzMTgzODE1
WjBoMQswCQYDVQQGEwJVUzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCU1lbmxvUGFy
azEMMAoGA1UECgwDT05GMRQwEgYDVQQLDAtFbmdpbmVlcmluZzEUMBIGA1UEAwwL
b25vcy1jb25maWcwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCZn2yJ
TuwzOI083/2/OaIwzQzvzuGBnp4ttZGxrSi0KdX7r19Vu2dI5eEDbSNNYa4+9xSw
T0lTf9u0/9i6mJYgejLNJU0CnAZDGxTLuxMHeGjPjeSpdVGuXq4qOj+kXKmdpATX
ZmyXwZlKgamh7/u66Wcrgn2y6eK1iCIvK+L0U+vwAa6NokUJwZlFtLBPmE0wwZxn
W9LMyWkzK0KANCsua3uVBSFclREsnM+jw4V2Gk9smXxfRV1HWkP7xsZuGi8NHq8A
1/JC1gE5CNLVsnD/ePHV3vHYFzVTu66Q9hDJLGpyu2cTpjUv44p7/3fQsaz3fYuT
31nAj5//l9ZHJwXRAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAGZfTp6qKTGZGrEl
bZWWKe8ilMqgZtcz7J93LOYk+l8nTEg5hIQ015mHY1+R+0F2gniciayjpVG2BChD
MOHfes0StdKY0nVHy83TpG0TsY76e//DSmekZwtm+OoxualpEOLW0PgKFEE8+PdJ
b/QlN8AyWJ3cvA7hDGlCrCNontLJS+W0VAPLDrFi/NeK0RpiQ6rI2U4B2jdGIrhw
AJD2FhJHDcdfBeR80KHiSFlhgSSMChKBrlzYw2vdeuSuAuuzTn88CzXKaIki56xQ
xm422D8l3cAo4W+GP6HGtwMl/UcI+WBMpa6yGtkaCaUA9v5EwKk+oceDlSnNnHS9
tWg4SMI=
-----END CERTIFICATE-----
`
