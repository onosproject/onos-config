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
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// GetTransaction returns response with the requested transaction
func (s Server) GetTransaction(ctx context.Context, req *admin.GetTransactionRequest) (*admin.GetTransactionResponse, error) {
	log.Infof("Received GetTransaction request: %+v", req)
	conf, err := s.transactionsStore.Get(ctx, req.ID)
	if err != nil {
		log.Warnf("GetTransaction %+v failed: %v", req, err)
		return nil, errors.Status(err).Err()
	}
	return &admin.GetTransactionResponse{Transaction: conf}, nil
}

// ListTransactions provides stream listing all transactions
func (s Server) ListTransactions(req *admin.ListTransactionsRequest, stream admin.TransactionService_ListTransactionsServer) error {
	log.Infof("Received ListTransactions request: %+v", req)
	transactions, err := s.transactionsStore.List(stream.Context())
	if err != nil {
		log.Warnf("ListTransactions %+v failed: %v", req, err)
		return errors.Status(err).Err()
	}
	for _, conf := range transactions {
		err := stream.Send(&admin.ListTransactionsResponse{Transaction: conf})
		if err != nil {
			log.Warnf("ListTransactions %+v failed: %v", req, err)
			return errors.Status(err).Err()
		}
	}
	return nil
}

// WatchTransactions provides stream with events representing transactions changes
func (s Server) WatchTransactions(req *admin.WatchTransactionsRequest, stream admin.TransactionService_WatchTransactionsServer) error {
	log.Infof("Received WatchTransactions request: %+v", req)
	var watchOpts []transaction.WatchOption
	if !req.Noreplay {
		watchOpts = append(watchOpts, transaction.WithReplay())
	}

	ch := make(chan configapi.TransactionEvent)
	if err := s.transactionsStore.Watch(stream.Context(), ch, watchOpts...); err != nil {
		log.Warnf("WatchTransactionsRequest %+v failed: %v", req, err)
		return errors.Status(err).Err()
	}

	if err := s.streamTransactions(stream, ch); err != nil {
		return errors.Status(err).Err()
	}
	return nil
}

func (s Server) streamTransactions(server admin.TransactionService_WatchTransactionsServer, ch chan configapi.TransactionEvent) error {
	for event := range ch {
		res := &admin.WatchTransactionsResponse{
			Event: event,
		}

		log.Debugf("Sending WatchTransactionsResponse %+v", res)
		if err := server.Send(res); err != nil {
			log.Warnf("WatchTransactionsResponse send %+v failed: %v", res, err)
			return err
		}
	}
	return nil
}
