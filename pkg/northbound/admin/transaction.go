// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	transaction "github.com/onosproject/onos-config/pkg/store/v2/transaction"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// GetTransaction returns response with the requested transaction
func (s Server) GetTransaction(ctx context.Context, req *admin.GetTransactionRequest) (*admin.GetTransactionResponse, error) {
	log.Infof("Received GetTransaction request: %+v", req)
	var t *configapi.Transaction
	var err error
	if req.Index > 0 { // if index is specified it takes precedence as lookup criteria
		t, err = s.transactionsStore.GetByIndex(ctx, req.Index)
	} else {
		t, err = s.transactionsStore.Get(ctx, req.ID)
	}
	if err != nil {
		log.Warnf("GetTransaction %+v failed: %v", req, err)
		return nil, errors.Status(err).Err()
	}
	return &admin.GetTransactionResponse{Transaction: t}, nil
}

// ListTransactions provides stream listing all transactions
func (s Server) ListTransactions(req *admin.ListTransactionsRequest, stream admin.TransactionService_ListTransactionsServer) error {
	log.Infof("Received ListTransactions request: %+v", req)
	transactions, err := s.transactionsStore.List(stream.Context())
	if err != nil {
		log.Warnf("ListTransactions %+v failed: %v", req, err)
		return errors.Status(err).Err()
	}
	for _, t := range transactions {
		err := stream.Send(&admin.ListTransactionsResponse{Transaction: t})
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

	if len(req.ID) > 0 {
		watchOpts = append(watchOpts, transaction.WithTransactionID(req.ID))
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
			TransactionEvent: event,
		}

		log.Debugf("Sending WatchTransactionsResponse %+v", res)
		if err := server.Send(res); err != nil {
			log.Warnf("WatchTransactionsResponse send %+v failed: %v", res, err)
			return err
		}
	}
	return nil
}
