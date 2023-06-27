// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package admin

import (
	"context"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	configuration "github.com/onosproject/onos-config/pkg/store/v2/configuration"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// GetConfiguration returns response with the requested configuration
func (s Server) GetConfiguration(ctx context.Context, req *admin.GetConfigurationRequest) (*admin.GetConfigurationResponse, error) {
	log.Infof("Received GetConfiguration request: %+v", req)
	conf, err := s.configurationsStore.Get(ctx, req.ConfigurationID)
	if err != nil {
		log.Warnf("GetConfiguration %+v failed: %v", req, err)
		return nil, errors.Status(err).Err()
	}
	return &admin.GetConfigurationResponse{Configuration: conf}, nil
}

// ListConfigurations provides stream listing all configurations
func (s Server) ListConfigurations(req *admin.ListConfigurationsRequest, stream admin.ConfigurationService_ListConfigurationsServer) error {
	log.Infof("Received ListConfigurations request: %+v", req)
	configurations, err := s.configurationsStore.List(stream.Context())
	if err != nil {
		log.Warnf("ListConfigurations %+v failed: %v", req, err)
		return errors.Status(err).Err()
	}
	for _, conf := range configurations {
		err := stream.Send(&admin.ListConfigurationsResponse{Configuration: conf})
		if err != nil {
			log.Warnf("ListConfigurations %+v failed: %v", req, err)
			return errors.Status(err).Err()
		}
	}
	return nil
}

// WatchConfigurations provides stream with events representing configuration changes
func (s Server) WatchConfigurations(req *admin.WatchConfigurationsRequest, stream admin.ConfigurationService_WatchConfigurationsServer) error {
	log.Infof("Received WatchConfigurations request: %+v", req)
	var watchOpts []configuration.WatchOption
	if !req.Noreplay {
		watchOpts = append(watchOpts, configuration.WithReplay())
	}

	if len(req.ConfigurationID) > 0 {
		watchOpts = append(watchOpts, configuration.WithConfigurationID(req.ConfigurationID))
	}

	ch := make(chan configapi.ConfigurationEvent)
	if err := s.configurationsStore.Watch(stream.Context(), ch, watchOpts...); err != nil {
		log.Warnf("WatchConfigurationsRequest %+v failed: %v", req, err)
		return errors.Status(err).Err()
	}

	if err := s.streamConfigurations(stream, ch); err != nil {
		return errors.Status(err).Err()
	}
	return nil
}

func (s Server) streamConfigurations(server admin.ConfigurationService_WatchConfigurationsServer, ch chan configapi.ConfigurationEvent) error {
	for event := range ch {
		res := &admin.WatchConfigurationsResponse{
			ConfigurationEvent: event,
		}

		log.Debugf("Sending WatchConfigurationsResponse %+v", res)
		if err := server.Send(res); err != nil {
			log.Warnf("WatchConfigurationsResponse send %+v failed: %v", res, err)
			return err
		}
	}
	return nil
}
