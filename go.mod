module github.com/onosproject/onos-config

go 1.13

require (
	github.com/atomix/api v0.0.0-20200219005318-0350f11bfcde
	github.com/atomix/go-client v0.0.0-20200218200323-6fd69e684d05
	github.com/atomix/go-framework v0.0.0-20200211010924-f3f12b63db0a
	github.com/atomix/go-local v0.0.0-20200211010611-c99e53e4c653
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/go-logfmt/logfmt v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.3
	github.com/google/uuid v1.1.1
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onosproject/config-models v0.0.0-20200214135352-e7b5a6a6c992
	github.com/onosproject/config-models/modelplugin/devicesim-1.0.0 v0.0.0-20200214110049-38f1117cd24a
	github.com/onosproject/config-models/modelplugin/stratum-1.0.0 v0.0.0-20200214111606-c266c76f882c
	github.com/onosproject/config-models/modelplugin/testdevice-1.0.0 v0.0.0-20200214110049-38f1117cd24a
	github.com/onosproject/config-models/modelplugin/testdevice-2.0.0 v0.0.0-20200214110049-38f1117cd24a
	github.com/onosproject/onos-test v0.0.0-20200225182923-ec7134d073e6
	github.com/onosproject/onos-topo v0.0.0-20200218171206-55029b503689
	github.com/openconfig/gnmi v0.0.0-20190823184014-89b2bf29312c
	github.com/openconfig/goyang v0.0.0-20200115183954-d0a48929f0ea
	github.com/openconfig/ygot v0.6.1-0.20200103195725-e3c44fa43926
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.6
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.12.0
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2 // indirect
	golang.org/x/sys v0.0.0-20200212091648-12a6c2dcc1e4 // indirect
	google.golang.org/genproto v0.0.0-20200212174721-66ed5ce911ce // indirect
	google.golang.org/grpc v1.27.1
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/client-go v0.0.0-20190620074045-585a16d2e773
	k8s.io/klog v1.0.0
)
