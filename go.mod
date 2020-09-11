module github.com/onosproject/onos-config

go 1.14

require (
	github.com/Pallinder/go-randomdata v1.2.0
	github.com/atomix/go-client v0.2.3
	cloud.google.com/go v0.43.0 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.3
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/onosproject/config-models/modelplugin/devicesim-1.0.0 v0.0.0-20200903015527-386223ad48bd
	github.com/onosproject/config-models/modelplugin/testdevice-1.0.0 v0.0.0-20200903015527-386223ad48bd
	github.com/onosproject/config-models/modelplugin/testdevice-2.0.0 v0.0.0-20200903015527-386223ad48bd
	github.com/onosproject/helmit v0.6.7
	github.com/onosproject/onos-lib-go v0.6.19
	github.com/onosproject/onos-test v0.6.2
	github.com/onosproject/onos-topo v0.6.17
	github.com/openconfig/gnmi v0.0.0-20190823184014-89b2bf29312c
	github.com/openconfig/goyang v0.0.0-20200115183954-d0a48929f0ea
	github.com/openconfig/ygot v0.6.1-0.20200103195725-e3c44fa43926
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.6
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	go.uber.org/multierr v1.4.0 // indirect
	golang.org/x/sys v0.0.0-20200212091648-12a6c2dcc1e4 // indirect
	golang.org/x/tools v0.0.0-20200113040837-eac381796e91 // indirect
	google.golang.org/genproto v0.0.0-20200212174721-66ed5ce911ce // indirect
	google.golang.org/grpc v1.31.1
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
	k8s.io/client-go v0.17.3
)

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20200229013735-71373c6105e3
