// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package path

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReplacePrefixes(t *testing.T) {
	assert.Equal(t, "/dingo/foo/bar/goo", RemovePathPrefixes("/dingo/p1:foo/p2:bar/goo"))
	assert.Equal(t, "dingo/foo/bar/goo", RemovePathPrefixes("dingo/p1:foo/p2:bar/goo"))
	assert.Equal(t, "/dingo/foo/bar/goo", RemovePathPrefixes("/p0:dingo/p1:foo/p2:bar/goo"))
	assert.Equal(t, "dingo/foo/bar/goo", RemovePathPrefixes("p0:dingo/p1:foo/p2:bar/goo"))
}

func TestRestoreIndexes(t *testing.T) {
	assert.Equal(t, "/p1:foo[a=42]/p2:bar[name=hey]/goo",
		RestoreIndexes("/p1:foo[a=*]/p2:bar[name=*]/goo", "/foo[a=42]/bar[name=hey]/goo"))

	assert.Equal(t, "/p1:foo[a=42,b=69]/p2:bar[name=hey]/goo",
		RestoreIndexes("/p1:foo[a=*,b=*]/p2:bar[name=*]/goo", "/foo[a=42,b=69]/bar[name=hey]/goo"))
}

var rwPathMap = ReadWritePathMap{
	"p1:foo/bar/gigi":             {},
	"p1:foo/bar/p2:goo/a":         {},
	"p1:foo/bar/p2:goo/z":         {},
	"p1:foo/bar/p2:boo[name=*]/a": {},
	"p1:foo/bar/p2:boo[name=*]/b": {},
	"p1:foo/bar/p3:dingo":         {},
}

var barePaths = StrippedPathMap(rwPathMap)

func TestFindPathFromModel(t *testing.T) {
	// Prefixed path should match
	exact, path, _, err := FindPathFromModel("p1:foo/bar/p2:goo/z", barePaths, rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, exact)
	assert.Equal(t, "p1:foo/bar/p2:goo/z", path)

	// Prefixed invalid path should produce error
	exact, _, _, err = FindPathFromModel("p1:foo/bar/p2:boo/z", barePaths, rwPathMap, false)
	assert.Error(t, err)
	assert.False(t, exact)

	// Bare path should match
	exact, path, _, err = FindPathFromModel("foo/bar/goo/z", barePaths, rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, exact)
	assert.Equal(t, "p1:foo/bar/p2:goo/z", path)

	// Partially prefixed path should also match
	exact, path, _, err = FindPathFromModel("p1:foo/bar/goo/z", barePaths, rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, exact)
	assert.Equal(t, "p1:foo/bar/p2:goo/z", path)

	// Partially prefixed and indexed path should also match
	exact, path, _, err = FindPathFromModel("p1:foo/bar/p2:boo[name=5]/b", barePaths, rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, exact)
	assert.Equal(t, "p1:foo/bar/p2:boo[name=5]/b", path)

	// Bare indexed path should also match
	exact, path, _, err = FindPathFromModel("foo/bar/boo[name=5]/a", barePaths, rwPathMap, false)
	assert.NoError(t, err)
	assert.True(t, exact)
	assert.Equal(t, "p1:foo/bar/p2:boo[name=5]/a", path)
}
