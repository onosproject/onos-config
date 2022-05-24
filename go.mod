module github.com/onosproject/onos-config

go 1.16

require (
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/atomix/atomix-go-client v0.6.2
	github.com/atomix/atomix-go-framework v0.10.1
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/onosproject/config-models/models/testdevice-2.0.x v0.0.0-20220402084627-ae31ddb22e46
	github.com/onosproject/helmit v0.6.19
	github.com/onosproject/onos-api/go v0.9.15
	github.com/onosproject/onos-lib-go v0.8.13
	github.com/onosproject/onos-ric-sdk-go v0.8.9
	github.com/onosproject/onos-test v0.6.6
	github.com/onosproject/onos-topo v0.9.3
	github.com/openconfig/gnmi v0.0.0-20210914185457-51254b657b7d
	github.com/openconfig/goyang v0.4.0
	github.com/prometheus/common v0.26.0
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	google.golang.org/grpc v1.41.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	k8s.io/client-go v0.22.1
)

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20200229013735-71373c6105e3

replace github.com/onosproject/onos-api/go => /Users/arastega/go/src/github.com/onosproject/onos-api/go
