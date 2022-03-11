// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/onosproject/helmit/pkg/benchmark"
	"github.com/onosproject/helmit/pkg/registry"
	"github.com/onosproject/onos-config/benchmark/gnmi"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	registry.RegisterBenchmarkSuite("gnmi", &gnmi.BenchmarkSuite{})
	benchmark.Main()
}
