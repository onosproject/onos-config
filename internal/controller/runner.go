package controller

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func RunTests(t *testing.T, path string, f func(*testing.T, *TestContext)) {
	NewTestRunner(t).Run(path, f)
}

func NewTestRunner(t *testing.T) *TestRunner {
	return &TestRunner{
		T: t,
	}
}

type TestRunner struct {
	*testing.T
}

func (r *TestRunner) Run(path string, f func(*testing.T, *TestContext)) {
	scanner, err := r.open(path)
	if !assert.NoError(r.T, err) {
		return
	}

	i := 0
	for scanner.Scan() {
		bytes := scanner.Bytes()
		name := fmt.Sprintf("%s-%d", r.T.Name(), i)
		ok := r.T.Run(name, func(t *testing.T) {
			var test test.TestCase
			assert.NoError(t, json.Unmarshal(bytes, &test))
			context := newTestContext(t, test)
			defer context.Finish()
			f(t, context)
		})
		if !ok {
			return
		}
	}
}

func (r *TestRunner) open(path string) (*bufio.Scanner, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

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
		return bufio.NewScanner(tarfile), nil
	default:
		return nil, errors.New("malformed test case file")
	}
}
