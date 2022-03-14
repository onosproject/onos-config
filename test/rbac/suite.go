// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test/utils/charts"
	"github.com/onosproject/onos-test/pkg/onostest"
)

type testSuite struct {
	test.Suite
}

// TestSuite is the onos-config HA test suite
type TestSuite struct {
	testSuite
	keycloakPassword string
}

func getKeycloakPassword() (string, error) {
	kubClient, err := kubernetes.New()
	if err != nil {
		return "", err
	}
	secrets, err := kubClient.CoreV1().Secrets().Get(context.Background(), onostest.SecretsName)
	if err != nil {
		return "", err
	}
	keycloakPassword := string(secrets.Object.Data["keycloak-password"])

	return keycloakPassword, nil
}

// SetupTestSuite sets up the onos-config RBAC test suite
func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	password, err := getKeycloakPassword()
	if err != nil {
		return err
	}
	s.keycloakPassword = password
	umbrella := charts.CreateUmbrellaRelease().
		Set("onos-config.openidc.issuer", "https://keycloak-dev.onlab.us/auth/realms/master").
		Set("onos-config.openpolicyagent.regoConfigMap", "onos-umbrella-opa-rbac").
		Set("onos-config.openpolicyagent.enabled", true)
	return umbrella.Install(true)
}
