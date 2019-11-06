#!/bin/sh

proto_imports=".:${GOPATH}/src/github.com/gogo/protobuf/protobuf:${GOPATH}/src/github.com/gogo/protobuf:${GOPATH}/src"

# admin.proto cannot be generated with fast marshaler/unmarshaler because it uses gnmi.ModelData
protoc -I=$proto_imports --gogo_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-config/api/admin,plugins=grpc:. api/admin/*.proto
protoc -I=$proto_imports --gogo_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mconfig/admin/admin.proto=github.com/onosproject/onos-config/api/admin,import_path=github.com/onosproject/onos-config/api/diags,plugins=grpc:. api/diags/*.proto

protoc -I=$proto_imports --gogofaster_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-config/api/types/change,plugins=grpc:. api/types/change/*.proto
protoc -I=$proto_imports --gogofaster_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-config/api/types/change/network,plugins=grpc:. api/types/change/network/*.proto
protoc -I=$proto_imports --gogofaster_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-config/api/types/change/device,plugins=grpc:. api/types/change/device/*.proto
protoc -I=$proto_imports --gogofaster_out=Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-config/api/types/snapshot,plugins=grpc:. api/types/snapshot/*.proto
protoc -I=$proto_imports --gogofaster_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-config/api/types/snapshot/network,plugins=grpc:. api/types/snapshot/network/*.proto
protoc -I=$proto_imports --gogofaster_out=Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,import_path=github.com/onosproject/onos-config/api/types/snapshot/device,plugins=grpc:. api/types/snapshot/device/*.proto
