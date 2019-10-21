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

package utils

import (
	"gotest.tools/assert"
	"testing"
)

func Test_Wildcard1(t *testing.T) {
	const wildcard1 = "/aa/*/cc/*/ee"
	pathRegexp1 := MatchWildcardRegexp(wildcard1)

	const path1 = "/aa/bb/cc/dd/ee"
	assert.Assert(t, pathRegexp1.MatchString(path1), "Expect match "+path1)

	const path2 = "/aa/bb/cc/dd/ee/ff"
	assert.Assert(t, pathRegexp1.MatchString(path2), "Expect match "+path2)

	const path3 = "/aa/q-q/cc/d_d/ee"
	assert.Assert(t, pathRegexp1.MatchString(path3), "Expect match "+path3)

	const path4 = "/aa/qq/rr/dd/ee"
	assert.Assert(t, !pathRegexp1.MatchString(path4), "Expect NO match "+path4)

	const path5 = "/aa/bb/cc/dd"
	assert.Assert(t, !pathRegexp1.MatchString(path5), "Expect NO match "+path5)
}

func Test_Wildcard2(t *testing.T) {
	const wildcard2 = `/ww/*/xx[name=*]/yy`
	pathRegexp2 := MatchWildcardRegexp(wildcard2)

	const path1 = "/ww/aa/xx[name=eth1]/yy"
	assert.Assert(t, pathRegexp2.MatchString(path1), "Expect match "+path1)

	const path2 = "/ww/test1:a_b-c/xx[name=eth2.100,top]/yy"
	assert.Assert(t, pathRegexp2.MatchString(path2), "Expect match "+path2)

	const path3 = "/ww/test1:aa/xx[name=eth2]/yy/zz"
	assert.Assert(t, pathRegexp2.MatchString(path3), "Expect match "+path3)

	const path4 = "/ww/test1:aa/xx[name1=eth1]/yy"
	assert.Assert(t, !pathRegexp2.MatchString(path4), "Expect NO match "+path4)

}

func Test_Wildcard3(t *testing.T) {
	const wildcard3 = `/ww/*/xx[name=*]/yy/zz[id=*]`
	pathRegexp3 := MatchWildcardRegexp(wildcard3)

	const path1 = "/ww/aa/xx[name=eth1]/yy/zz[id=2]"
	assert.Assert(t, pathRegexp3.MatchString(path1), "Expect match "+path1)

	const path2 = "/ww/test1:a_b-c/xx[name=eth2.100,top]/yy/zz[id=22]"
	assert.Assert(t, pathRegexp3.MatchString(path2), "Expect match "+path2)

	const path3 = "/ww/test1:aa/xx[name=eth2]/yy/zz[id=2_2]/aa"
	assert.Assert(t, pathRegexp3.MatchString(path3), "Expect match "+path3)

	const path4 = "/ww/test1:aa/xx[name1=eth1]/yy/zz"
	assert.Assert(t, !pathRegexp3.MatchString(path4), "Expect NO match "+path4)

}

func Test_Wildcard4(t *testing.T) {
	const wildcard4 = `/ww/.../yy/*/bb`
	pathRegexp4 := MatchWildcardRegexp(wildcard4)

	const path1 = "/ww/aa/xx[name=eth1]/yy/aa/bb"
	assert.Assert(t, pathRegexp4.MatchString(path1), "Expect match "+path1)

	const path2 = "/ww/test1:a_b-c/xx[name=eth2.100,top]/yy/aa/bb"
	assert.Assert(t, pathRegexp4.MatchString(path2), "Expect match "+path2)

	const path3 = "/ww/test1:aa/xx[name=eth2]/yy/zz"
	assert.Assert(t, !pathRegexp4.MatchString(path3), "Expect NO match "+path3)

	const path4 = "/ww/test1:aa/xx[name1=eth1]/yy"
	assert.Assert(t, !pathRegexp4.MatchString(path4), "Expect NO match "+path4)

}

func Test_Wildcard_Change1(t *testing.T) {
	const wildcard1 = `cha?ge-*`
	pathRegexp1 := MatchWildcardChNameRegexp(wildcard1)

	const chID1 = "change-2"
	assert.Assert(t, pathRegexp1.MatchString(chID1), "Expect match "+chID1)

	const chID2 = "charge-123456"
	assert.Assert(t, pathRegexp1.MatchString(chID2), "Expect match "+chID2)

	const chID3 = "channge-2"
	assert.Assert(t, !pathRegexp1.MatchString(chID3), "Expect not match "+chID3)

	const chID4 = "cha+ge-2"
	assert.Assert(t, !pathRegexp1.MatchString(chID4), "Expect not match "+chID4)
}

func Test_Wildcard_Change2(t *testing.T) {
	const wildcard2 = `cha*ge-??`
	pathRegexp2 := MatchWildcardChNameRegexp(wildcard2)

	const chID1 = "change-22"
	assert.Assert(t, pathRegexp2.MatchString(chID1), "Expect match "+chID1)

	const chID2 = "charge-123456"
	assert.Assert(t, pathRegexp2.MatchString(chID2), "Expect match "+chID2)

	const chID3 = "channge-23"
	assert.Assert(t, pathRegexp2.MatchString(chID3), "Expect not match "+chID3)

	const chID4 = "cha+++ge-23"
	assert.Assert(t, !pathRegexp2.MatchString(chID4), "Expect not match "+chID4)

	const chID5 = "channge-++"
	assert.Assert(t, !pathRegexp2.MatchString(chID5), "Expect not match "+chID5)
}
