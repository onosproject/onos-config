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

package southbound

import (
	"context"
	"github.com/openconfig/gnmi/client"
)

// CacheClientInterface : interface to hide struct dependency on client.CacheClient. Can be overridden by tests.
type BaseClientInterface interface {
	Subscribe(context.Context, client.Query, ...string) error
}

// GnmiCacheClientFactory : Default CacheClient creation.
var GnmiBaseClientFactory = func() BaseClientInterface {
	return gnmiBaseClientImpl{
		&client.BaseClient{},
	}
}

// GnmiCacheClientImpl : shim to hide Gnmi CacheClient type dependency.
type gnmiBaseClientImpl struct {
	c *client.BaseClient
}

// Subscribe : default implementation to subscribe via the cached client.
func (c gnmiBaseClientImpl) Subscribe(ctx context.Context, q client.Query, types ...string) error {
	return c.c.Subscribe(ctx, q, types...)
}
