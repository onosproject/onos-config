package integration

import "testing"

func TestIntegrationTest(t *testing.T) {

}

func init() {
	Registry.Register("test-integration-test", TestIntegrationTest)
}
