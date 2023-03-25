// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	helmit "github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test"
	"github.com/onosproject/onos-test/pkg/onostest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestSuite is the onos-config HA test suite
type TestSuite struct {
	test.Suite
	keycloakPassword string
	simulator        string
}

func (s *TestSuite) getKeycloakPassword(ctx context.Context) string {
	secrets, err := s.CoreV1().Secrets(s.Namespace()).Get(ctx, onostest.SecretsName, metav1.GetOptions{})
	s.NoError(err)
	keycloakPassword := string(secrets.Data["keycloak-password"])
	return keycloakPassword
}

// SetupSuite sets up the onos-config RBAC test suite
func (s *TestSuite) SetupSuite(ctx context.Context) {
	s.keycloakPassword = s.getKeycloakPassword(ctx)
	err := s.InstallUmbrella().
		Set("onos-config.openidc.issuer", "https://keycloak-dev.onlab.us/auth/realms/master").
		Set("onos-config.openpolicyagent.regoConfigMap", "onos-umbrella-opa-rbac").
		Set("onos-config.openpolicyagent.enabled", true).
		Wait().
		Do(ctx)
	s.NoError(err)
}

// TearDownSuite tears down the test suite
func (s *TestSuite) TearDownSuite(ctx context.Context) {
	s.NoError(s.Helm().Uninstall("onos-umbrella").Do(ctx))
}

func (s *TestSuite) SetupTest(ctx context.Context) {
	s.simulator = s.SetupRandomSimulator(ctx, true)
}

func (s *TestSuite) TearDownTest(ctx context.Context) {
	s.TearDownSimulator(ctx, s.simulator)
}

var _ helmit.SetupSuite = (*TestSuite)(nil)
var _ helmit.SetupTest = (*TestSuite)(nil)
var _ helmit.TearDownTest = (*TestSuite)(nil)
