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

package utils

import (
	"context"
	"github.com/cenkalti/backoff"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"sync"
	"time"
)

// RetryingUnaryClientInterceptor returns a UnaryClientInterceptor that retries requests
func RetryingUnaryClientInterceptor() func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return backoff.Retry(func() error {
			if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
				if isRetryable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
			return nil
		}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
	}
}

// RetryingStreamClientInterceptor returns a ClientStreamInterceptor that retries both requests and responses
func RetryingStreamClientInterceptor(duration time.Duration) func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		stream := &retryingClientStream{
			ctx:      ctx,
			buffer:   make([]interface{}, 0, 1),
			duration: duration,
			newStream: func(ctx context.Context) (grpc.ClientStream, error) {
				return streamer(ctx, desc, cc, method, opts...)
			},
		}
		return stream, stream.retryStream()
	}
}

type retryingClientStream struct {
	ctx       context.Context
	stream    grpc.ClientStream
	duration  time.Duration
	mu        sync.RWMutex
	buffer    []interface{}
	newStream func(ctx context.Context) (grpc.ClientStream, error)
	closed    bool
}

func (s *retryingClientStream) setStream(stream grpc.ClientStream) {
	s.mu.Lock()
	s.stream = stream
	s.mu.Unlock()
}

func (s *retryingClientStream) getStream() grpc.ClientStream {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stream
}

func (s *retryingClientStream) Context() context.Context {
	return s.ctx
}

func (s *retryingClientStream) CloseSend() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	if err := s.getStream().CloseSend(); err != nil {
		return err
	}
	return nil
}

func (s *retryingClientStream) Header() (metadata.MD, error) {
	return s.getStream().Header()
}

func (s *retryingClientStream) Trailer() metadata.MD {
	return s.getStream().Trailer()
}

func (s *retryingClientStream) SendMsg(m interface{}) error {
	s.mu.Lock()
	s.buffer = append(s.buffer, m)
	s.mu.Unlock()
	if err := s.getStream().SendMsg(m); err != nil {
		return err
	}
	return nil
}

func (s *retryingClientStream) RecvMsg(m interface{}) error {
	if err := s.getStream().RecvMsg(m); err != nil {
		return backoff.Retry(func() error {
			if err := s.retryStream(); err != nil {
				if isRetryable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
			if err := s.getStream().RecvMsg(m); err != nil {
				if isRetryable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
			return nil
		}, backoff.NewExponentialBackOff())
	}
	return nil
}

func (s *retryingClientStream) retryStream() error {
	return backoff.Retry(func() error {
		stream, err := s.newStream(context.Background())
		if err != nil {
			return err
		}

		s.mu.RLock()
		buffer := s.buffer
		closed := s.closed
		s.mu.RUnlock()
		for _, m := range buffer {
			if err := stream.SendMsg(m); err != nil {
				if isRetryable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
		}

		if closed {
			if err := stream.CloseSend(); err != nil {
				if isRetryable(err) {
					return err
				}
				return backoff.Permanent(err)
			}
		}

		s.setStream(stream)
		return nil
	}, backoff.NewConstantBackOff(s.duration))
}

func isRetryable(err error) bool {
	st := status.Code(err)
	if st == codes.Unavailable || st == codes.Unknown {
		return true
	}
	return false
}
