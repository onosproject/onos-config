module github.com/onosproject/onos-config

go 1.12

require (
	github.com/atomix/atomix-go-client v0.0.0-20191005004655-89d80d892bdf
	github.com/atomix/atomix-go-local v0.0.0-20191005004851-a9403b577637
	github.com/atomix/atomix-go-node v0.0.0-20191005004913-89f218752fa8
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/gogo/protobuf v1.2.2-0.20190723190241-65acae22fc9d
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onosproject/onos-control v0.0.0-20190715190020-706a2ee0d37b // indirect
	github.com/onosproject/onos-topo v0.0.0-20191003071657-8bcd1fd812c5
	github.com/openconfig/gnmi v0.0.0-20180912164834-33a1865c3029
	github.com/openconfig/goyang v0.0.0-20190408185115-e8b0ed2cbb0c
	github.com/openconfig/ygot v0.5.1-0.20190427030428-68346f97239f
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	google.golang.org/grpc v1.23.1
	gotest.tools v2.2.0+incompatible
	k8s.io/klog v0.3.3
)

replace github.com/atomix/atomix-go-node => ../atomix-go-node
