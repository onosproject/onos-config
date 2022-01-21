// Copyright 2022-present Open Networking Foundation.
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

package admin

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/config/admin"
)

// GetTransaction returns response with the requested configuration
func (s Server) GetTransaction(ctx context.Context, req *admin.GetTransactionRequest) (*admin.GetTransactionResponse, error) {
	return nil, nil
}

// ListTransactions provides stream listing all configurations
func (s Server) ListTransactions(req *admin.ListTransactionsRequest, stream admin.TransactionService_ListTransactionsServer) error {
	return nil
}

// WatchTransactions provides stream with events representing configuration changes
func (s Server) WatchTransactions(req *admin.WatchTransactionsRequest, stream admin.TransactionService_WatchTransactionsServer) error {
	return nil
}
