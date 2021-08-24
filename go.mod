module github.com/onosproject/onos-config

go 1.15

require (
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/atomix/atomix-go-client v0.5.20
	github.com/atomix/atomix-go-framework v0.7.0
	github.com/atomix/go-client v0.4.1
	github.com/bshuster-repo/logrus-logstash-hook v1.0.0 // indirect
	github.com/bugsnag/bugsnag-go v2.1.1+incompatible // indirect
	github.com/bugsnag/panicwrap v1.3.2 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/garyburd/redigo v1.6.2 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.1.2
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/onosproject/config-models/modelplugin/devicesim-1.0.0 v0.6.34
	github.com/onosproject/config-models/modelplugin/testdevice-1.0.0 v0.6.34
	github.com/onosproject/config-models/modelplugin/testdevice-2.0.0 v0.6.34
	github.com/onosproject/helmit v0.6.12
	github.com/onosproject/onos-api/go v0.7.87
	github.com/onosproject/onos-config-model v0.4.8
	github.com/onosproject/onos-lib-go v0.7.13
	github.com/onosproject/onos-test v0.6.5
	github.com/openconfig/gnmi v0.0.0-20200617225440-d2b4e6a45802
	github.com/openconfig/goyang v0.2.5
	github.com/openconfig/ygot v0.11.2
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/smartystreets/assertions v1.0.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/yvasiyarov/go-metrics v0.0.0-20150112132944-c25f46c4b940 // indirect
	github.com/yvasiyarov/gorelic v0.0.7 // indirect
	github.com/yvasiyarov/newrelic_platform_go v0.0.0-20160601141957-9c099fbc30e9 // indirect
	google.golang.org/grpc v1.37.0
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
	k8s.io/client-go v0.17.3
)

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20200229013735-71373c6105e3
