package model

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

func RunTests[S, C any](t *testing.T, path string, f func(*testing.T, TestCase[S, C])) {
	t.Helper()

	file, err := os.Open(path)
	if !assert.NoError(t, err) {
		return
	}

	reader, err := NewReader[S, C](file)
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

func NewReader[S, C any](file io.Reader) (*Reader[S, C], error) {
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
		return &Reader[S, C]{
			scanner: scanner,
			hashes:  make(map[string]bool),
		}, nil
	default:
		return nil, errors.New("malformed test case file")
	}
}

type Reader[S, C any] struct {
	scanner *bufio.Scanner
	hashes  map[string]bool
}

func (r *Reader[S, C]) Next() (TestCase[S, C], error) {
	for {
		testCase, ok, err := r.read()
		if err != nil {
			return testCase, err
		} else if ok {
			return testCase, nil
		}
	}
}

func (r *Reader[S, C]) unique(bytes []byte) bool {
	hash := md5.Sum(bytes)
	key := hex.EncodeToString(hash[:])
	_, ok := r.hashes[key]
	if !ok {
		r.hashes[key] = true
		return true
	}
	return false
}

func (r *Reader[S, C]) read() (TestCase[S, C], bool, error) {
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
