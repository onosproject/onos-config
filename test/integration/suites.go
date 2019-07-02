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

package integration

import (
	"github.com/onosproject/onos-config/test/runner"
)

var (
	//AllTests is an example of a test suite
	AllTests = runner.NewTestSuite("alltests")
	//SomeTests is an example of a test suite
	SomeTests = runner.NewTestSuite("sometests")
)

func init(){
	//example of registering groups
	Registry.RegisterTestSuite(*AllTests)
	Registry.RegisterTestSuite(*SomeTests)
}