#!/bin/sh

# SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

mockgen -source=pkg/controller/controller.go -destination=pkg/controller/controller_mock_test.go -package=controller
mockgen -source=pkg/controller/activator.go -destination=pkg/controller/activator_mock_test.go -package=controller
mockgen -source=pkg/controller/filter.go -destination=pkg/controller/filter_mock_test.go -package=controller
mockgen -source=pkg/controller/partitioner.go -destination=pkg/controller/partitioner_mock_test.go -package=controller -aux_files github.com/onosproject/onos-config/pkg/controller=pkg/controller/partitioner.go
