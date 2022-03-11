// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"gotest.tools/assert"
	"testing"
)

func Test_Wildcard1(t *testing.T) {
	const wildcard1 = "/aa/*/cc/*/ee"
	pathRegexpNonExact := MatchWildcardRegexp(wildcard1, false)
	pathRegexpExact := MatchWildcardRegexp(wildcard1, true)

	const path1 = "/aa/bb/cc/dd/ee"
	assert.Assert(t, pathRegexpNonExact.MatchString(path1), "Expect match "+path1)
	assert.Assert(t, pathRegexpExact.MatchString(path1), "Expect match "+path1)

	const path2 = "/aa/bb/cc/dd/ee/ff"
	assert.Assert(t, pathRegexpNonExact.MatchString(path2), "Expect match "+path2)
	assert.Assert(t, !pathRegexpExact.MatchString(path2), "Expect NO match "+path2)

	const path3 = "/aa/q-q/cc/d_d/ee"
	assert.Assert(t, pathRegexpNonExact.MatchString(path3), "Expect match "+path3)
	assert.Assert(t, pathRegexpExact.MatchString(path3), "Expect match "+path3)

	const path4 = "/aa/qq/rr/dd/ee"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path4), "Expect NO match "+path4)
	assert.Assert(t, !pathRegexpExact.MatchString(path4), "Expect NO match "+path4)

	const path5 = "/aa/bb/cc/dd"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path5), "Expect NO match "+path5)
	assert.Assert(t, !pathRegexpExact.MatchString(path5), "Expect NO match "+path5)
}

func Test_Wildcard2(t *testing.T) {
	const wildcard2 = `/ww/*/xx[name=*]/yy`
	pathRegexpNonExact := MatchWildcardRegexp(wildcard2, false)
	pathRegexpExact := MatchWildcardRegexp(wildcard2, true)

	const path1 = "/ww/aa/xx[name=eth1]/yy"
	assert.Assert(t, pathRegexpNonExact.MatchString(path1), "Expect match "+path1)
	assert.Assert(t, pathRegexpExact.MatchString(path1), "Expect match "+path1)

	const path2 = "/ww/test1:a_b-c/xx[name=eth2.100,top]/yy"
	assert.Assert(t, pathRegexpNonExact.MatchString(path2), "Expect match "+path2)
	assert.Assert(t, pathRegexpExact.MatchString(path2), "Expect match "+path2)

	const path3 = "/ww/test1:aa/xx[name=eth2]/yy/zz"
	assert.Assert(t, pathRegexpNonExact.MatchString(path3), "Expect match "+path3)
	assert.Assert(t, !pathRegexpExact.MatchString(path3), "Expect NO match "+path3)

	const path4 = "/ww/test1:aa/xx[name1=eth1]/yy"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path4), "Expect NO match "+path4)
	assert.Assert(t, !pathRegexpExact.MatchString(path4), "Expect NO match "+path4)

	const path5 = "/ww/test1:aa/qq[name=eth1]/yy"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path5), "Expect NO match "+path5)
	assert.Assert(t, !pathRegexpExact.MatchString(path5), "Expect NO match "+path5)

}

func Test_Wildcard3(t *testing.T) {
	const wildcard3 = `/ww/*/xx[name=*]/yy/zz[id=*]`
	pathRegexpNonExact := MatchWildcardRegexp(wildcard3, false)
	pathRegexpExact := MatchWildcardRegexp(wildcard3, true)

	const path1 = "/ww/aa/xx[name=eth1]/yy/zz[id=2]"
	assert.Assert(t, pathRegexpNonExact.MatchString(path1), "Expect match "+path1)
	assert.Assert(t, pathRegexpExact.MatchString(path1), "Expect match "+path1)

	const path2 = "/ww/test1:a_b-c/xx[name=eth2.100,top]/yy/zz[id=22]"
	assert.Assert(t, pathRegexpNonExact.MatchString(path2), "Expect match "+path2)
	assert.Assert(t, pathRegexpExact.MatchString(path2), "Expect match "+path2)

	const path3 = "/ww/test1:aa/xx[name=eth2]/yy/zz[id=2_2]/aa"
	assert.Assert(t, pathRegexpNonExact.MatchString(path3), "Expect match "+path3)
	assert.Assert(t, !pathRegexpExact.MatchString(path3), "Expect NO match "+path3)

	const path4 = "/ww/test1:aa/xx[name1=eth1]/yy/zz"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path4), "Expect NO match "+path4)
	assert.Assert(t, !pathRegexpExact.MatchString(path4), "Expect NO match "+path4)

}

func Test_Wildcard4(t *testing.T) {
	const wildcard4 = `/ww/.../yy/*/bb`
	pathRegexpNonExact := MatchWildcardRegexp(wildcard4, false)
	pathRegexpExact := MatchWildcardRegexp(wildcard4, true)

	const path1 = "/ww/aa/xx[name=eth1]/yy/aa/bb"
	assert.Assert(t, pathRegexpNonExact.MatchString(path1), "Expect match "+path1)
	assert.Assert(t, pathRegexpExact.MatchString(path1), "Expect match "+path1)

	const path1a = "/ww/aa/xx[name=eth1]/yy/aa/bb/cc"
	assert.Assert(t, pathRegexpNonExact.MatchString(path1a), "Expect match "+path1a)
	assert.Assert(t, !pathRegexpExact.MatchString(path1a), "Expect NO match "+path1a)

	const path2 = "/ww/test1:a_b-c/xx[name=eth2.100,top]/yy/qq/bb"
	assert.Assert(t, pathRegexpNonExact.MatchString(path2), "Expect match "+path2)
	assert.Assert(t, pathRegexpExact.MatchString(path2), "Expect match "+path2)

	const path3 = "/ww/test1:aa/xx[name=eth2]/yy/zz"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path3), "Expect NO match "+path3)
	assert.Assert(t, !pathRegexpExact.MatchString(path3), "Expect NO match "+path3)

	const path4 = "/ww/test1:aa/xx[name1=eth1]/yy"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path4), "Expect NO match "+path4)
	assert.Assert(t, !pathRegexpExact.MatchString(path4), "Expect NO match "+path4)

}
func Test_Wildcard5(t *testing.T) {
	const wildcard5 = `/ww/.../yy/a*/bb`
	pathRegexpNonExact := MatchWildcardRegexp(wildcard5, false)
	pathRegexpExact := MatchWildcardRegexp(wildcard5, true)

	const path1 = "/ww/aa/xx[name=eth1]/yy/aa/bb"
	assert.Assert(t, pathRegexpNonExact.MatchString(path1), "Expect match "+path1)
	assert.Assert(t, pathRegexpExact.MatchString(path1), "Expect match "+path1)

	const path1a = "/ww/aa/xx[name=eth1]/yy/qq/bb/cc"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path1a), "Expect NO match "+path1a)
	assert.Assert(t, !pathRegexpExact.MatchString(path1a), "Expect NO match "+path1a)

	const path2 = "/ww/test1:a_b-c/xx[name=eth2.100,top]/yy/ad/bb"
	assert.Assert(t, pathRegexpNonExact.MatchString(path2), "Expect match "+path2)
	assert.Assert(t, pathRegexpExact.MatchString(path2), "Expect match "+path2)

	const path3 = "/ww/test1:aa/xx[name=eth2]/yy/zz"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path3), "Expect NO match "+path3)
	assert.Assert(t, !pathRegexpExact.MatchString(path3), "Expect NO match "+path3)

	const path4 = "/ww/test1:aa/xx[name1=eth1]/yy"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path4), "Expect NO match "+path4)
	assert.Assert(t, !pathRegexpExact.MatchString(path4), "Expect NO match "+path4)

	const path5 = "/ww/test1:aa/xx[name1=eth1]/yy/ad[name=test]/bb"
	assert.Assert(t, !pathRegexpNonExact.MatchString(path5), "Expect NO match "+path5)
	assert.Assert(t, !pathRegexpExact.MatchString(path5), "Expect NO match "+path5)

}

func Test_Wildcard_Change1(t *testing.T) {
	const wildcard1 = `cha?ge-*`
	pathRegexpNonExact := MatchWildcardChNameRegexp(wildcard1, false)
	pathRegexpExact := MatchWildcardChNameRegexp(wildcard1, true)

	const chID1 = "change-2"
	assert.Assert(t, pathRegexpNonExact.MatchString(chID1), "Expect match "+chID1)
	assert.Assert(t, pathRegexpExact.MatchString(chID1), "Expect match "+chID1)

	const chID2 = "charge-123456"
	assert.Assert(t, pathRegexpNonExact.MatchString(chID2), "Expect match "+chID2)
	assert.Assert(t, pathRegexpExact.MatchString(chID2), "Expect match "+chID2)

	const chID3 = "channge-2"
	assert.Assert(t, !pathRegexpNonExact.MatchString(chID3), "Expect not match "+chID3)
	assert.Assert(t, !pathRegexpExact.MatchString(chID3), "Expect not match "+chID3)

	const chID4 = "cha+ge-2"
	assert.Assert(t, !pathRegexpNonExact.MatchString(chID4), "Expect not match "+chID4)
	assert.Assert(t, !pathRegexpExact.MatchString(chID4), "Expect not match "+chID4)
}

func Test_Wildcard_Change2(t *testing.T) {
	const wildcard2 = `cha*ge-??`
	pathRegexpExact := MatchWildcardChNameRegexp(wildcard2, true)
	pathRegexpNonExact := MatchWildcardChNameRegexp(wildcard2, false)

	const chID1 = "change-22"
	assert.Assert(t, pathRegexpExact.MatchString(chID1), "Expect match "+chID1)
	assert.Assert(t, pathRegexpNonExact.MatchString(chID1), "Expect match "+chID1)

	const chID2 = "charge-123456" // Should not match exact
	assert.Assert(t, !pathRegexpExact.MatchString(chID2), "Expect not match "+chID2)
	assert.Assert(t, pathRegexpNonExact.MatchString(chID2), "Expect match "+chID2)

	const chID3 = "channge-23"
	assert.Assert(t, pathRegexpExact.MatchString(chID3), "Expect match "+chID3)
	assert.Assert(t, pathRegexpNonExact.MatchString(chID3), "Expect match "+chID3)

	const chID4 = "chann"
	assert.Assert(t, !pathRegexpExact.MatchString(chID4), "Expect not match "+chID4)
	assert.Assert(t, !pathRegexpNonExact.MatchString(chID4), "Expect not match "+chID4)

	const chID5 = "hann-ge5"
	assert.Assert(t, !pathRegexpExact.MatchString(chID5), "Expect not match "+chID5)
	assert.Assert(t, !pathRegexpNonExact.MatchString(chID5), "Expect not match "+chID5)

	const chID6 = "cha+++ge-23"
	assert.Assert(t, !pathRegexpExact.MatchString(chID6), "Expect not match "+chID6)
	assert.Assert(t, !pathRegexpNonExact.MatchString(chID6), "Expect not match "+chID6)

	const chID7 = "channge-++" // + is not a legal character
	assert.Assert(t, !pathRegexpExact.MatchString(chID7), "Expect not match "+chID7)

	const chID8 = "channge-.." // Dot is a legal character
	assert.Assert(t, pathRegexpExact.MatchString(chID8), "Expect match "+chID8)
}
