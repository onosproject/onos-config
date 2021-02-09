// Copyright 2021-present Open Networking Foundation.
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

package rbac

import (
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/stretchr/testify/assert"
	"testing"
)

func samplePathValues() []*devicechange.PathValue {
	sampleValues := make([]*devicechange.PathValue, 0)
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/a/b/c/e", // Should match /a/?/c and /a/b/c
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/a/b/c/f", // Should match /a/?/c and /a/b/c
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/a/b/d", // should not match /a/?/c or /a/b/c
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/a/a/c", // Should match /a/?/c
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/q/r/s",
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/a/b", // Not matched
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/x/y/z", // Matched
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "/x/y/z1", // Matched
		Value: devicechange.NewTypedValueBool(true),
	})
	sampleValues = append(sampleValues, &devicechange.PathValue{
		Path:  "a/x/y/z", // Not matched - even with non-exact wildcard, we always match the start of the path
		Value: devicechange.NewTypedValueBool(true),
	})

	return sampleValues
}

func Test_FilterGetByRoles(t *testing.T) {
	cache := setupRbacCache()
	sampleValues := samplePathValues()

	filtered, err := cache.FilterGetByRbacRoles(sampleValues, []string{group1ID, group3ID})
	assert.NoError(t, err)
	assert.True(t, filtered != nil)
	assert.Equal(t, 5, len(filtered))
	// Should always be in this order
	assert.Equal(t, "/a/a/c", filtered[0].Path)
	assert.Equal(t, "/a/b/c/e", filtered[1].Path)
	assert.Equal(t, "/a/b/c/f", filtered[2].Path)
	assert.Equal(t, "/x/y/z", filtered[3].Path)
	assert.Equal(t, "/x/y/z1", filtered[4].Path)

}

func Test_buildGroupRoleQuery(t *testing.T) {
	cache := setupRbacCache()

	groupSearch1 := buildGroupRoleQuery([]string{group1ID})
	assert.NotNil(t, groupSearch1)
	assert.Equal(t, `/rbac/group\[groupid=(group\-1)]/role\[roleid=.*]/roleid`, groupSearch1.String())
	for rbacPath := range cache.(*rbacCache).rbacMap {
		switch rbacPath {
		case
			"/rbac/group[groupid=group-1]/role[roleid=role-1]/roleid",
			"/rbac/group[groupid=group-1]/role[roleid=role-2]/roleid":
			assert.True(t, groupSearch1.MatchString(rbacPath))
		default:
			assert.False(t, groupSearch1.MatchString(rbacPath))
		}
	}

	groupSearch2 := buildGroupRoleQuery([]string{group1ID, group3ID})
	assert.NotNil(t, groupSearch2)
	assert.Equal(t, `/rbac/group\[groupid=(group\-1|group\-3)]/role\[roleid=.*]/roleid`, groupSearch2.String())
	for rbacPath := range cache.(*rbacCache).rbacMap {
		switch rbacPath {
		case
			"/rbac/group[groupid=group-1]/role[roleid=role-1]/roleid",
			"/rbac/group[groupid=group-1]/role[roleid=role-2]/roleid",
			"/rbac/group[groupid=group-3]/role[roleid=role-3]/roleid":
			assert.True(t, groupSearch2.MatchString(rbacPath))
		default:
			assert.False(t, groupSearch2.MatchString(rbacPath), rbacPath)
		}
	}

	groupSearchEmpty := buildGroupRoleQuery([]string{})
	assert.NotNil(t, groupSearchEmpty)
	assert.Equal(t, `/rbac/group\[groupid=()]/role\[roleid=.*]/roleid`, groupSearchEmpty.String())
	for rbacPath := range cache.(*rbacCache).rbacMap {
		switch rbacPath {
		default:
			assert.False(t, groupSearchEmpty.MatchString(rbacPath), rbacPath)
		}
	}
}

func Test_buildRoleQuery(t *testing.T) {
	cache := setupRbacCache()

	roleSearch := buildRoleQuery([]string{role1ID, role2ID})
	assert.NotNil(t, roleSearch)
	assert.Equal(t, `/rbac/role\[roleid=(role\-1|role\-2)]/permission/noun`, roleSearch.String())
	for rbacPath := range cache.(*rbacCache).rbacMap {
		switch rbacPath {
		case
			"/rbac/role[roleid=role-1]/permission/noun",
			"/rbac/role[roleid=role-2]/permission/noun":
			assert.True(t, roleSearch.MatchString(rbacPath), rbacPath)
		default:
			assert.False(t, roleSearch.MatchString(rbacPath), rbacPath)
		}
	}
}

func Test_findRolesForGroups(t *testing.T) {
	cache := setupRbacCache()

	roleIDs := cache.(*rbacCache).findRolesForGroups([]string{group1ID, group3ID})
	assert.NotNil(t, roleIDs)
	assert.Equal(t, 3, len(roleIDs))
	assert.Equal(t, role1ID, roleIDs[0]) // because they are sorted
	assert.Equal(t, role2ID, roleIDs[1]) // because they are sorted
	assert.Equal(t, role3ID, roleIDs[2]) // because they are sorted
}

func Test_findNounsForRoles(t *testing.T) {
	cache := setupRbacCache()
	nouns := cache.(*rbacCache).findNounsForRoles([]string{role1ID, role2ID})
	assert.Equal(t, 3, len(nouns))
	assert.Equal(t, "/a/b/c", nouns[0]) // because they are sorted
	assert.Equal(t, "/d/e/f", nouns[1])
	assert.Equal(t, "/g/h/i", nouns[2])
}
