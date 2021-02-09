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
	"github.com/onosproject/onos-config/pkg/utils"
	"regexp"
	"sort"
	"strings"
)

func (r *rbacCache) FilterGetByRbacRoles(getResults []*devicechange.PathValue,
	groupIds []string) ([]*devicechange.PathValue, error) {

	roleIDs := r.findRolesForGroups(groupIds)
	permittedNouns := r.findNounsForRoles(roleIDs)
	filteredSetMap := make(map[string]*devicechange.PathValue)
	for _, noun := range permittedNouns {
		regex := utils.MatchWildcardRegexp(noun, false)
		for _, pathValue := range getResults {
			if _, alreadyAdded := filteredSetMap[pathValue.GetPath()]; !alreadyAdded {
				if regex.MatchString(pathValue.GetPath()) {
					filteredSetMap[pathValue.GetPath()] = pathValue
				}
			}
		}
	}
	filteredSet := make([]*devicechange.PathValue, 0, len(filteredSetMap))
	for _, f := range filteredSetMap {
		filteredSet = append(filteredSet, f)
	}
	sort.Slice(filteredSet, func(i, j int) bool {
		return filteredSet[i].Path < filteredSet[j].Path
	})
	log.Infof("RBAC: Filtered %d paths to %d for groups %v", len(getResults), len(filteredSet), groupIds)
	return filteredSet, nil
}

func (r *rbacCache) findRolesForGroups(groupIDs []string) []string {
	groupSearch := buildGroupRoleQuery(groupIDs)
	roleIdsMap := make(map[string]interface{}) // To eliminate duplicates
	for rbacPath, rbacValue := range r.rbacMap {
		if groupSearch.MatchString(rbacPath) {
			roleName := (*devicechange.TypedString)(rbacValue).String()
			if _, ok := roleIdsMap[roleName]; !ok {
				roleIdsMap[roleName] = struct{}{}
			}
		}
	}
	roleIds := make([]string, 0, len(roleIdsMap))
	for rID := range roleIdsMap {
		roleIds = append(roleIds, rID)
	}
	sort.Slice(roleIds, func(i, j int) bool {
		return roleIds[i] < roleIds[j]
	})
	return roleIds
}

func (r *rbacCache) findNounsForRoles(roleIDs []string) []string {
	// Now go through the role definitions to extract nouns
	// Because it's Get() any permission will do - has to be CONFIG (type) though
	roleSearch := buildRoleQuery(roleIDs)
	permittedNounsMap := make(map[string]interface{}) //Avoids duplicates
	for rbacPath, rbacValue := range r.rbacMap {
		if roleSearch.MatchString(rbacPath) {
			roleNouns := (*devicechange.TypedLeafListString)(rbacValue).List()
			typeQuery := strings.Replace(rbacPath, "noun", "type", 1)
			if typeName, ok := r.rbacMap[typeQuery]; ok && string(typeName.Bytes) == "CONFIG" {
				for _, n := range roleNouns {
					if _, ok := permittedNounsMap[n]; !ok {
						permittedNounsMap[n] = struct{}{}
					}
				}
			}
		}
	}
	permittedNouns := make([]string, 0, len(permittedNounsMap))
	for n := range permittedNounsMap {
		permittedNouns = append(permittedNouns, n)
	}
	sort.Slice(permittedNouns, func(i, j int) bool {
		return permittedNouns[i] < permittedNouns[j]
	})
	return permittedNouns
}

func buildGroupRoleQuery(groupIds []string) *regexp.Regexp {
	groupQuery := strings.Builder{}
	groupQuery.WriteString(`/rbac/group\[groupid=(`)
	first := true
	for _, gID := range groupIds {
		if first {
			first = false
		} else {
			groupQuery.WriteString("|")
		}
		groupQuery.WriteString(strings.ReplaceAll(gID, "-", "\\-"))
	}
	groupQuery.WriteString(`)]/role\[roleid=.*]/roleid`)
	return regexp.MustCompile(groupQuery.String())
}

func buildRoleQuery(roleIDs []string) *regexp.Regexp {
	roleQuery := strings.Builder{}
	roleQuery.WriteString(`/rbac/role\[roleid=(`)
	first := true
	for _, rID := range roleIDs {
		if first {
			first = false
		} else {
			roleQuery.WriteString("|")
		}
		roleQuery.WriteString(strings.ReplaceAll(rID, "-", "\\-"))
	}
	roleQuery.WriteString(`)]/permission/noun`)
	return regexp.MustCompile(roleQuery.String())
}
