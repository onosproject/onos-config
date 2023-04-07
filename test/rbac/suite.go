// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"github.com/onosproject/helmit/pkg/helm"
	helmit "github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test"
	"github.com/onosproject/onos-test/pkg/onostest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestSuite is the onos-config RBAC test suite.
type TestSuite struct {
	test.Suite
	keycloakPassword string
	umbrella         *helm.Release
	simulator        *helm.Release
}

func (s *TestSuite) getKeycloakPassword() string {
	secrets, err := s.CoreV1().Secrets(s.Namespace()).Get(s.Context(), onostest.SecretsName, metav1.GetOptions{})
	s.NoError(err)
	keycloakPassword := string(secrets.Data["keycloak-password"])
	return keycloakPassword
}

// SetupSuite sets up the onos-config RBAC test suite
func (s *TestSuite) SetupSuite() {
	s.keycloakPassword = s.getKeycloakPassword()
	release, err := s.InstallUmbrella().
		Set("onos-config.openidc.issuer", "https://keycloak-dev.onlab.us/auth/realms/master").
		Set("onos-config.openpolicyagent.regoConfigMap", "onos-umbrella-opa-rbac").
		Set("onos-config.openpolicyagent.enabled", true).
		Wait().
		Get(s.Context())
	s.NoError(err)
	s.umbrella = release
}

// TearDownSuite tears down the test suite
func (s *TestSuite) TearDownSuite() {
	s.NoError(s.Helm().Uninstall(s.umbrella.Name).Do(s.Context()))
}

// SetupTest sets up simulators for tests
func (s *TestSuite) SetupTest() {
	s.simulator = s.SetupRandomSimulator(true)
}

// TearDownTest tears down simulators for tests
func (s *TestSuite) TearDownTest() {
	s.TearDownSimulator(s.simulator.Name)
}

var _ helmit.SetupSuite = (*TestSuite)(nil)
var _ helmit.SetupTest = (*TestSuite)(nil)
var _ helmit.TearDownTest = (*TestSuite)(nil)
