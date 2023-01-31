// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test/config"
	"github.com/onosproject/onos-config/test/rbac"
	"github.com/onosproject/onos-config/test/scaling"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	registry.RegisterTestSuite("config", &config.TestSuite{})
	registry.RegisterTestSuite("rbac", &rbac.TestSuite{})
	registry.RegisterTestSuite("scaling", &scaling.TestSuite{})

	test.Main()
}
