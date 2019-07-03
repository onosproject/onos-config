// Copyright 2019-present Open Networking Foundation.
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

package runner

import (
	"errors"
	"os"
	"sort"
	"testing"
)

// NewRegistry returns a pointer to a new TestRegistry
func NewRegistry() *TestRegistry {
	return &TestRegistry{
		tests:      make(map[string]Test),
		TestSuites: make(map[string]TestSuite),
	}
}

//NewTestSuite returns a pointer to a new TestSuite
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		name: name,
		tests: make(map[string]Test),
	}
}

// Test is a test function
type Test func(t *testing.T)


// TestRegistry contains a mapping of named test groups
type TestRegistry struct {
	tests      map[string]Test
	TestSuites map[string]TestSuite
}

//TestSuite to run multiple tests
type TestSuite struct {
	name string
	tests map[string]Test
}

//RegisterTest registers a test to the registry
func (r *TestRegistry) RegisterTest(name string, test Test, suites []*TestSuite) {
	r.tests[name] = test

	for _, suite := range suites {
		//fmt.Println("Registering test: ", name, "on suite: ",suite.name)
		suite.registerTest(name, test)
	}
}

//RegisterTest registers a test to a test group
func (r *TestSuite) registerTest(name string, test Test) {
	r.tests[name] = test
}

//RegisterTestSuite registers test suite into the registry
func (r *TestRegistry) RegisterTestSuite(testGroup TestSuite) {
	r.TestSuites[testGroup.name] = testGroup
}

// GetTestNames returns a slice of test names
func (r *TestRegistry) GetTestNames() []string {
	names := make([]string, 0, len(r.tests))
	for name := range r.tests {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}

// GetTestNames returns a slice of test names
func (r *TestSuite) GetTestNames() []string {
	names := make([]string, 0, len(r.tests))
	for name := range r.tests {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}

// GetTestSuiteNames returns a slice of test names
func (r *TestRegistry) GetTestSuiteNames() []string {
	names := make([]string, 0, len(r.TestSuites))
	for name := range r.TestSuites {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}

// TestRunner runs integration tests
type TestRunner struct {
	Registry *TestRegistry
}


// RunTests Runs the tests
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

	// Hack to enable verbose testing.
	os.Args = []string{
		os.Args[0],
		"-test.v",
	}

	// Run the integration tests via the testing package.
	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
	return nil
}


// RunTestSuites Runs the tests groups
func (r *TestRunner) RunTestSuites(args []string) error {
	tests := make([]testing.InternalTest, 0, len(args))
	if len(args) > 0 {
		for _, name := range args {
			testSuite, ok := r.Registry.TestSuites[name]
			if !ok {
				return errors.New("unknown test suite" + name)
			}

			testNames := []string{}

			for testName := range testSuite.tests {
				testNames = append(testNames,testName)
			}
			r.RunTests(testNames)
		}
	} else {
		return nil
	}

	// Hack to enable verbose testing.
	os.Args = []string{
		os.Args[0],
		"-test.v",
	}

	// Run the integration tests via the testing package.
	testing.Main(func(_, _ string) (bool, error) { return true, nil }, tests, nil, nil)
	return nil
}
