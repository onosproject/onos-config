module github.com/onosproject/onos-config

go 1.12

require (
	github.com/atomix/atomix-k8s-controller v0.0.0-20190620084759-d5e65f7fbf68
	github.com/docker/docker v1.13.1
	github.com/fatih/color v1.7.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onosproject/onos-topo v0.0.0-20190806004156-537a9862c203
	github.com/openconfig/gnmi v0.0.0-20180912164834-33a1865c3029
	github.com/openconfig/goyang v0.0.0-20190408185115-e8b0ed2cbb0c
	github.com/openconfig/ygot v0.5.1-0.20190427030428-68346f97239f
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.3.0
	google.golang.org/grpc v1.22.1
	gopkg.in/yaml.v1 v1.0.0-20140924161607-9f9df34309c0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.0.0-20190620073856-dcce3486da33
	k8s.io/apiextensions-apiserver v0.0.0-20190325193600-475668423e9f
	k8s.io/apimachinery v0.0.0-20190802060556-6fa4771c83b3
	k8s.io/client-go v0.0.0-20190620074045-585a16d2e773
	k8s.io/klog v0.3.3
)

replace github.com/onosproject/onos-topo => ../onos-topo
