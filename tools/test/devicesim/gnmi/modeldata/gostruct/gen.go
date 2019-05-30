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

package gostruct

//go:generate sh -c "go get -u github.com/openconfig/ygot; (cd $GOPATH/src/github.com/openconfig/ygot && go get -t -d ./...); go get -u github.com/openconfig/public; go get -u github.com/YangModels/yang; cd $GOPATH/src && go run github.com/openconfig/ygot/generator/generator.go -generate_fakeroot -output_file github.com/google/gnxi/gnmi/modeldata/gostruct/generated.go -package_name gostruct -exclude_modules ietf-interfaces -path github.com/openconfig/public,github.com/YangModels/yang github.com/openconfig/public/release/models/interfaces/openconfig-interfaces.yang github.com/openconfig/public/release/models/openflow/openconfig-openflow.yang github.com/openconfig/public/release/models/platform/openconfig-platform.yang github.com/openconfig/public/release/models/system/openconfig-system.yang"
