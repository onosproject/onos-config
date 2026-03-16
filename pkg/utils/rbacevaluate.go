// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"strings"
)

const semicolon = ";" // From grpcinterceptors.go in onos-lib-go

// TemporaryEvaluate - simple evaluation of rules until OpenPolicyAgent is added
// This is so that aether-config can be deployed to the cloud in 2021 Q1 with simple RBAC
// It applies to Set (gnmi) and CompactChanges(admin) and RollbackNetworkChange(admin)
// TODO replace the following with fine grained RBAC using OpenPolicyAgent Rego in 2021 Q2
func TemporaryEvaluate(md metautils.NiceMD) error {
	adminGroups := os.Getenv("ADMINGROUPS")
	var match bool
	for _, g := range strings.Split(md.Get("groups"), semicolon) {
		if strings.Contains(adminGroups, g) {
			match = true
			break
		}
	}
	if !match {
		return status.Errorf(codes.Unauthenticated, "Set allowed only for %s", adminGroups)
	}
	return nil
}

func GetAllowedAdminGroups(md metautils.NiceMD) string {
	var groups string
	for _, g := range strings.Split(md.Get("groups"), semicolon) {
		if g == "AetherROCAdmin" {
			return "AllGroups"
		} else {
			groups = groups + semicolon + g
		}
	}
	return groups
}

func IsAllowToDoSet(allowedGroups, prefixString string, deleteElem []*gnmi.Path, update, replace []*gnmi.Update) bool {
	type void struct{}
	var zeroByte void
	enterpriseSet := make(map[string]void)

	if len(prefixString) > 5 {
		entFromPrefix := getRequestedEnterprisesFromPrefixString(prefixString)
		for _, ent := range entFromPrefix {
			enterpriseSet[ent] = zeroByte
		}
	}
	if len(update) > 0 {
		entFromUpdate := getRequestedEnterprisesFromUpdate(update)
		for _, ent := range entFromUpdate {
			enterpriseSet[ent] = zeroByte
		}
	}
	if len(deleteElem) > 0 {
		entFromDelete := getRequestedEnterprisesFromDelete(deleteElem)
		for _, ent := range entFromDelete {
			enterpriseSet[ent] = zeroByte
		}
	}
	if len(replace) > 0 {
		entFromReplace := getRequestedEnterprisesFromReplace(replace)
		for _, ent := range entFromReplace {
			enterpriseSet[ent] = zeroByte
		}
	}
	allowedGroupList := strings.Split(allowedGroups, semicolon)
	delete(enterpriseSet, "")

	var allowed bool
	for ent := range enterpriseSet {
		for _, agl := range allowedGroupList {
			allowed = false
			if agl == ent {
				allowed = true
				break
			}
		}
		if !allowed {
			return allowed
		}
	}
	return allowed
}

func getEnterpriseFromElemString(elemString string) string {
	var allowedEnterprises string
	for _, x := range strings.Split(elemString, "elem:") {
		if strings.Contains(x, "enterprise-id") {
			for _, y := range strings.Fields(x) {
				if strings.Contains(y, "value") {
					allowedEnterprises = allowedEnterprises + semicolon + strings.ReplaceAll(strings.ReplaceAll(
						strings.ReplaceAll(y, "}", ""), "value:", ""), "\"", "")
				}
			}
		}
	}
	return allowedEnterprises
}

func getRequestedEnterprisesFromPrefixString(prefixString string) []string {
	n := getEnterpriseFromElemString(prefixString)
	return strings.Split(n, semicolon)
}

func getRequestedEnterprisesFromUpdate(update []*gnmi.Update) []string {
	var n string
	for _, x := range update {
		n = n + getEnterpriseFromElemString(x.GetPath().String())
	}
	return strings.Split(n, semicolon)
}

func getRequestedEnterprisesFromDelete(delete []*gnmi.Path) []string {
	var n string
	for _, x := range delete {
		n = n + getEnterpriseFromElemString(x.String())
	}
	return strings.Split(n, semicolon)
}

func getRequestedEnterprisesFromReplace(replace []*gnmi.Update) []string {
	var n string
	for _, x := range replace {
		n = n + getEnterpriseFromElemString(x.GetPath().String())
	}
	return strings.Split(n, semicolon)
}
