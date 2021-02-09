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
	"fmt"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	assert "github.com/stretchr/testify/assert"
	"testing"
)

const (
	role1ID = "role-1"
	role2ID = "role-2"
	role3ID = "role-3"

	group1ID = "group-1"
	group2ID = "group-2"
	group3ID = "group-3"

	groupIDPath          = "/rbac/group[groupid=%s]/groupid"
	groupDescPath        = "/rbac/group[groupid=%s]/description"
	groupRoleRefPath     = "/rbac/group[groupid=%s]/role[roleid=%s]/roleid"
	groupRoleRefDescPath = "/rbac/group[groupid=%s]/role[roleid=%s]/description"

	roleIDPath   = "/rbac/role[roleid=%s]/roleid"
	rolePathDesc = "/rbac/role[roleid=%s]/description"
	rolePathNoun = "/rbac/role[roleid=%s]/permission/noun"
	rolePathOp   = "/rbac/role[roleid=%s]/permission/operation"
	rolePathType = "/rbac/role[roleid=%s]/permission/type"
)

// 3 roles referred to in 3 groups
//         group-1    group-2     group-3
// role-1     ✔
// role-2     ✔          ✔
// role-3                ✔           ✔

func setupRbacCache() Cache {
	cache := &rbacCache{
		rbacMap: make(devicechange.TypedValueMap),
	}
	cache.rbacMap[fmt.Sprintf(groupIDPath, group1ID)] = devicechange.NewTypedValueString(group1ID)
	cache.rbacMap[fmt.Sprintf(groupDescPath, group1ID)] = devicechange.NewTypedValueString(fmt.Sprintf("Group %s", group1ID))
	cache.rbacMap[fmt.Sprintf(groupRoleRefPath, group1ID, role1ID)] = devicechange.NewTypedValueString(role1ID)
	cache.rbacMap[fmt.Sprintf(groupRoleRefDescPath, group1ID, role1ID)] = devicechange.NewTypedValueString(fmt.Sprintf("%s in %s", role1ID, group1ID))
	cache.rbacMap[fmt.Sprintf(groupRoleRefPath, group1ID, role2ID)] = devicechange.NewTypedValueString(role2ID)
	cache.rbacMap[fmt.Sprintf(groupRoleRefDescPath, group1ID, role2ID)] = devicechange.NewTypedValueString(fmt.Sprintf("%s in %s", role2ID, group2ID))

	cache.rbacMap[fmt.Sprintf(groupIDPath, group2ID)] = devicechange.NewTypedValueString(group2ID)
	cache.rbacMap[fmt.Sprintf(groupDescPath, group2ID)] = devicechange.NewTypedValueString(fmt.Sprintf("Group %s", group2ID))
	cache.rbacMap[fmt.Sprintf(groupRoleRefPath, group2ID, role2ID)] = devicechange.NewTypedValueString(role2ID)
	cache.rbacMap[fmt.Sprintf(groupRoleRefDescPath, group2ID, role2ID)] = devicechange.NewTypedValueString(fmt.Sprintf("%s in %s", role2ID, group2ID))
	cache.rbacMap[fmt.Sprintf(groupRoleRefPath, group2ID, role3ID)] = devicechange.NewTypedValueString(role3ID)
	cache.rbacMap[fmt.Sprintf(groupRoleRefDescPath, group2ID, role3ID)] = devicechange.NewTypedValueString(fmt.Sprintf("%s in %s", role3ID, group2ID))

	cache.rbacMap[fmt.Sprintf(groupIDPath, group3ID)] = devicechange.NewTypedValueString(group3ID)
	cache.rbacMap[fmt.Sprintf(groupDescPath, group3ID)] = devicechange.NewTypedValueString(fmt.Sprintf("Group %s", group3ID))
	cache.rbacMap[fmt.Sprintf(groupRoleRefPath, group3ID, role3ID)] = devicechange.NewTypedValueString(role3ID)
	cache.rbacMap[fmt.Sprintf(groupRoleRefDescPath, group3ID, role3ID)] = devicechange.NewTypedValueString(fmt.Sprintf("%s in %s", role3ID, group3ID))

	cache.rbacMap[fmt.Sprintf(roleIDPath, role1ID)] = devicechange.NewTypedValueString(role1ID)
	cache.rbacMap[fmt.Sprintf(rolePathDesc, role1ID)] = devicechange.NewTypedValueString(fmt.Sprintf("Role %s", role1ID))
	cache.rbacMap[fmt.Sprintf(rolePathNoun, role1ID)] = devicechange.NewLeafListStringTv([]string{"/a/b/c", "/d/e/f"})
	cache.rbacMap[fmt.Sprintf(rolePathOp, role1ID)] = devicechange.NewTypedValueString("READ")
	cache.rbacMap[fmt.Sprintf(rolePathType, role1ID)] = devicechange.NewTypedValueString("CONFIG")

	cache.rbacMap[fmt.Sprintf(roleIDPath, role2ID)] = devicechange.NewTypedValueString(role2ID)
	cache.rbacMap[fmt.Sprintf(rolePathDesc, role2ID)] = devicechange.NewTypedValueString(fmt.Sprintf("Role %s", role2ID))
	cache.rbacMap[fmt.Sprintf(rolePathNoun, role2ID)] = devicechange.NewLeafListStringTv([]string{"/a/b/c", "/d/e/f", "/g/h/i"})
	cache.rbacMap[fmt.Sprintf(rolePathOp, role2ID)] = devicechange.NewTypedValueString("ALL")
	cache.rbacMap[fmt.Sprintf(rolePathType, role2ID)] = devicechange.NewTypedValueString("CONFIG")

	cache.rbacMap[fmt.Sprintf(roleIDPath, role3ID)] = devicechange.NewTypedValueString(role3ID)
	cache.rbacMap[fmt.Sprintf(rolePathNoun, role3ID)] = devicechange.NewLeafListStringTv([]string{"/a/*/c", "/x/y/z"})
	cache.rbacMap[fmt.Sprintf(rolePathOp, role3ID)] = devicechange.NewTypedValueString("ALL")
	cache.rbacMap[fmt.Sprintf(rolePathType, role3ID)] = devicechange.NewTypedValueString("CONFIG")

	return cache
}

func Test_handleChanges(t *testing.T) {
	cache := setupRbacCache()
	assert.Equal(t, 30, len(cache.(*rbacCache).rbacMap))

	changeValues := make([]*devicechange.ChangeValue, 0)
	changeValues = append(changeValues, &devicechange.ChangeValue{
		Path:    fmt.Sprintf("/rbac/group[groupid=%s]", group1ID),
		Value:   nil,
		Removed: true,
	})

	cache.(*rbacCache).handleChanges(changeValues)
	assert.Equal(t, 24, len(cache.(*rbacCache).rbacMap))
	_, ok1 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-1]/groupid"]
	assert.False(t, ok1)
	_, ok2 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-1]/description"]
	assert.False(t, ok2)
	_, ok3 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-1]/role[roleid=role-11]/roleid"]
	assert.False(t, ok3)
	_, ok4 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-1]/role[roleid=role-11]/description"]
	assert.False(t, ok4)
	_, ok5 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-1]/role[roleid=role-2]/roleid"]
	assert.False(t, ok5)
	_, ok6 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-1]/role[roleid=role-2]/description"]
	assert.False(t, ok6)

	changeValues2 := make([]*devicechange.ChangeValue, 0)
	changeValues2 = append(changeValues2, &devicechange.ChangeValue{
		Path:    fmt.Sprintf("/rbac/group[groupid=%s]/role[roleid=%s]", group2ID, role2ID),
		Value:   nil,
		Removed: true,
	})
	cache.(*rbacCache).handleChanges(changeValues2)
	assert.Equal(t, 22, len(cache.(*rbacCache).rbacMap))
	_, ok7 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-2]/role[roleid=role-2]/roleid"]
	assert.False(t, ok7)
	_, ok8 := cache.(*rbacCache).rbacMap["/rbac/group[groupid=group-2]/role[roleid=role-2]/description"]
	assert.False(t, ok8)

	changeValues3 := make([]*devicechange.ChangeValue, 0)
	changeValues3 = append(changeValues3, &devicechange.ChangeValue{
		Path:    fmt.Sprintf("/rbac/role[roleid=%s]", role2ID),
		Value:   nil,
		Removed: true,
	})
	cache.(*rbacCache).handleChanges(changeValues3)
	assert.Equal(t, 17, len(cache.(*rbacCache).rbacMap))
	_, ok30 := cache.(*rbacCache).rbacMap["/rbac/role[roleid=role-2]/roleid"]
	assert.False(t, ok30)
	_, ok31 := cache.(*rbacCache).rbacMap["/rbac/role[roleid=role-2]/description"]
	assert.False(t, ok31)
	_, ok32 := cache.(*rbacCache).rbacMap["/rbac/role[roleid=role-2]/permission/noun"]
	assert.False(t, ok32)
	_, ok33 := cache.(*rbacCache).rbacMap["/rbac/role[roleid=role-2]/permission/operation"]
	assert.False(t, ok33)
	_, ok34 := cache.(*rbacCache).rbacMap["/rbac/role[roleid=role-2]/permission/type"]
	assert.False(t, ok34)

}
