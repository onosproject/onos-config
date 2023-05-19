# SPDX-FileCopyrightText: 2019-present Open Networking Foundation <info@opennetworking.org>
#
# SPDX-License-Identifier: Apache-2.0

SHELL = bash -e -o pipefail

export CGO_ENABLED=1
export GO111MODULE=on

helmit-config: # @HELP run helmit gnmi tests locally
	kubectl delete ns test 2>/dev/null || exit 0
	helmit test --create-namespace --namespace test --no-teardown ./config/... --set onos-config.global.image.tag=latest --timeout 40m


helmit-rbac: # @HELP run helmit gnmi tests locally
	helmit test -n test ./onos-config-tests --suite rbac --secret keycloak-password=${keycloak_password} --no-teardown --timeout 40m

