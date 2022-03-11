// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
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
