package model

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

const (
	defaultCountEstimate = 5000000
	defaultFPRate        = .05
)

func RunTests[S, C any](t *testing.T, path string, f func(*testing.T, TestCase[S, C])) {
	t.Helper()

	file, err := os.Open(path)
	if !assert.NoError(t, err) {
		return
	}

	reader, err := NewReader[S, C](file, defaultCountEstimate)
	if !assert.NoError(t, err) {
		return
	}

	i := 0
	//start := 106735
	for {
		i++
		testCase, err := reader.Next()
		if err == io.EOF {
			break
		}
		if !assert.NoError(t, err) {
			break
		}
		//if i < start {
		//	continue
		//}
		testName := fmt.Sprintf("%s%d", t.Name(), i)
		ok := t.Run(testName, func(t *testing.T) {
			f(t, testCase)
		})
		if !ok {
			break
		}
	}
}

func NewReader[S, C any](file io.Reader, countEstimate uint) (*Reader[S, C], error) {
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
			filter:  bloom.NewWithEstimates(countEstimate, defaultFPRate),
		}, nil
	default:
		return nil, errors.New("malformed test case file")
	}
}

type Reader[S, C any] struct {
	scanner *bufio.Scanner
	filter  *bloom.BloomFilter
}

func (r *Reader[S, C]) Next() (TestCase[S, C], error) {
	var testCase TestCase[S, C]
	if !r.scanner.Scan() {
		return testCase, io.EOF
	}
	bytes := r.scanner.Bytes()
	if err := json.Unmarshal(bytes, &testCase); err != nil {
		return testCase, err
	}
	return testCase, nil
}
