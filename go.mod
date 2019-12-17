module github.com/onosproject/onos-config

go 1.12

require (
	github.com/atomix/atomix-go-client v0.0.0-20191127222459-36981d701c6e
	github.com/atomix/atomix-go-local v0.0.0-20191108201451-9131cc896ed6
	github.com/atomix/atomix-go-node v0.0.0-20191108201428-59c0962b63c8
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.3.1
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onosproject/onos-test v0.0.0-20191220005421-b22ebab3a08f
	github.com/onosproject/onos-topo v0.0.0-20191113170912-88eeee89f4eb
	github.com/openconfig/gnmi v0.0.0-20190823184014-89b2bf29312c
	github.com/openconfig/goyang v0.0.0-20190924211109-064f9690516f
	github.com/openconfig/ygot v0.6.0
	github.com/pkg/errors v0.8.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.4.0
	go.uber.org/zap v1.12.0
	google.golang.org/grpc v1.25.0
	gotest.tools v2.2.0+incompatible
	k8s.io/klog v0.3.3
)

replace github.com/onosproject/onos-test => ../onos-test
