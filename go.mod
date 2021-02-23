module github.com/onosproject/onos-config

go 1.14

require (
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/atomix/go-client v0.4.1
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/docker v1.13.1 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.1.2
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/onosproject/config-models/modelplugin/devicesim-1.0.0 v0.0.0-20201130213019-492043aed0df
	github.com/onosproject/config-models/modelplugin/testdevice-1.0.0 v0.0.0-20201130213019-492043aed0df
	github.com/onosproject/config-models/modelplugin/testdevice-2.0.0 v0.0.0-20201130213019-492043aed0df
	github.com/onosproject/helmit v0.6.10
	github.com/onosproject/onos-api/go v0.7.2
	github.com/onosproject/onos-lib-go v0.7.2
	github.com/onosproject/onos-test v0.6.4
	github.com/openconfig/gnmi v0.0.0-20200617225440-d2b4e6a45802
	github.com/openconfig/goyang v0.2.1
	github.com/openconfig/ygot v0.8.12
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.6.2 // indirect
	github.com/stretchr/testify v1.5.1
	go.uber.org/multierr v1.4.0 // indirect
	golang.org/x/tools v0.0.0-20200522201501-cb1345f3a375 // indirect
	google.golang.org/grpc v1.33.2
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	k8s.io/client-go v0.17.3
)

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20200229013735-71373c6105e3
