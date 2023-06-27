// SPDX-FileCopyrightText: 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

// RunTests runs the tests in the given model-based test case file
// The parameters S and C define the test state and test context structs respectively.
// For each test in the file, the given function f will be called with the TestCase
// populated with the initial state, transitions, and the context in which those
// transitions occurred.
func RunTests[S, C any](t *testing.T, path string, f func(*testing.T, TestCase[S, C])) {
	t.Helper()

	file, err := os.Open(path)
	if !assert.NoError(t, err) {
		return
	}

	reader, err := NewTestReader[S, C](file)
	if !assert.NoError(t, err) {
		return
	}

	i := 0
	for {
		i++
		testCase, err := reader.Next()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			break
		}
		testName := fmt.Sprintf("%s%d", t.Name(), i)
		ok := t.Run(testName, func(t *testing.T) {
			f(t, testCase)
		})
		if !ok {
			break
		}
	}
}

// TestCase is a model-based test case containing the state and changes expected in a single step of the model.
type TestCase[S, C any] struct {
	Context     C `json:"context"`
	State       S `json:"state"`
	Transitions S `json:"transitions"`
}

// NewTestReader creates a new TestReader that reads the given Reader.
func NewTestReader[S, C any](file io.Reader) (*TestReader[S, C], error) {
	gzfile, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}

	tarfile := tar.NewReader(gzfile)

	header, err := tarfile.Next()
	if err != nil {
		return nil, err
	}

	switch header.Typeflag {
	case tar.TypeReg:
		scanner := bufio.NewScanner(tarfile)
		return &TestReader[S, C]{
			scanner: scanner,
			hashes:  make(map[string]bool),
		}, nil
	default:
		return nil, errors.NewInternal("malformed test file")
	}
}

// TestReader reads and deduplicates TestCases from an input file.
type TestReader[S, C any] struct {
	scanner *bufio.Scanner
	hashes  map[string]bool
}

// Next gets the next TestCase in the file.
// If the test case is a duplicate of a test case that was already read from the file, the reader
// will continue reading until it either finds a unique test case or reaches the end of the file.
func (r *TestReader[S, C]) Next() (TestCase[S, C], error) {
	for {
		testCase, ok, err := r.read()
		if err != nil {
			return testCase, err
		} else if ok {
			return testCase, nil
		}
	}
}

func (r *TestReader[S, C]) unique(bytes []byte) bool {
	hash := md5.Sum(bytes)
	key := hex.EncodeToString(hash[:])
	_, ok := r.hashes[key]
	if !ok {
		r.hashes[key] = true
		return true
	}
	return false
}

func (r *TestReader[S, C]) read() (TestCase[S, C], bool, error) {
	var testCase TestCase[S, C]
	if !r.scanner.Scan() {
		return testCase, false, io.EOF
	}
	bytes := r.scanner.Bytes()
	if !r.unique(bytes) {
		return testCase, false, nil
	}
	if err := json.Unmarshal(bytes, &testCase); err != nil {
		return testCase, false, err
	}
	return testCase, true, nil
}
