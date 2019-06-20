package runner

import (
	"errors"
	"github.com/stretchr/testify/suite"
	"sort"
	"testing"
)

func NewRegistry() *TestRegistry {
	return &TestRegistry{
		tests: make(map[string]Test),
	}
}

// TestRegistry contains a mapping of named tests
type TestRegistry struct {
	suite.SetupTestSuite
	tests map[string]Test
}

// Register registers a named test
func (r *TestRegistry) Register(name string, test Test) {
	r.tests[name] = test
}

// GetNames returns a slice of test names
func (r *TestRegistry) GetNames() []string {
	names := make([]string, 0, len(r.tests))
	for name, _ := range r.tests {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}

// Test is a test function
type Test func(t *testing.T)

// TestRunner runs integration tests
type TestRunner struct {
	Registry *TestRegistry
}

// Runs the tests
func (r *TestRunner) RunTests(args []string) error {
	tests := make([]testing.InternalTest, 0, len(args))
	if len(args) > 0 {
		for _, name := range args {
			test, ok := r.Registry.tests[name]
			if !ok {
				return errors.New("unknown test " + name)
			}
			tests = append(tests, testing.InternalTest{
				Name: name,
				F:    test,
			})
		}
	} else {
		for name, test := range r.Registry.tests {
			tests = append(tests, testing.InternalTest{
				Name: name,
				F:    test,
			})
		}
	}
	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
	return nil
}
