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
