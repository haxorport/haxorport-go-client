package service

import (
	"fmt"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
)


type TunnelService struct {
	tunnelRepo port.TunnelRepository
	logger     port.Logger
}


func NewTunnelService(tunnelRepo port.TunnelRepository, logger port.Logger) *TunnelService {
	return &TunnelService{
		tunnelRepo: tunnelRepo,
		logger:     logger,
	}
}


func (s *TunnelService) CreateHTTPTunnel(localPort int, subdomain string, auth *model.TunnelAuth) (*model.Tunnel, error) {
	s.logger.Info("Creating HTTP tunnel for local port %d with subdomain %s", localPort, subdomain)


	tunnelConfig := model.TunnelConfig{
		Type:      model.TunnelTypeHTTP,
		LocalPort: localPort,
		Subdomain: subdomain,
		Auth:      auth,
	}

	// Register tunnel
	tunnel, err := s.tunnelRepo.Register(tunnelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to register HTTP tunnel: %v", err)
	}

	s.logger.Info("HTTP tunnel created successfully with URL: %s", tunnel.URL)

	return tunnel, nil
}


func (s *TunnelService) CreateTCPTunnel(config model.TunnelConfig) (*model.Tunnel, error) {

	if config.LocalAddr == "" {
		config.LocalAddr = "127.0.0.1"
	}


	s.logger.Info("Creating TCP tunnel from local port %d to %s:%d with remote port %d", 
		config.LocalPort, config.LocalAddr, config.LocalPort, config.RemotePort)


	config.Type = model.TunnelTypeTCP

	// Register tunnel
	tunnel, err := s.tunnelRepo.Register(config)
	if err != nil {
		return nil, fmt.Errorf("failed to register TCP tunnel: %v", err)
	}

	s.logger.Info("TCP tunnel created successfully with remote port: %d", tunnel.RemotePort)

	return tunnel, nil
}


func (s *TunnelService) CloseTunnel(tunnelID string) error {
	s.logger.Info("Closing tunnel with ID: %s", tunnelID)


	tunnel, err := s.tunnelRepo.GetByID(tunnelID)
	if err != nil {
		return fmt.Errorf("tunnel not found: %v", err)
	}

	s.logger.Info("Closing tunnel %s with type %s", tunnelID, tunnel.Config.Type)


	if err := s.tunnelRepo.Unregister(tunnelID); err != nil {
		return fmt.Errorf("failed to remove tunnel: %v", err)
	}

	s.logger.Info("Tunnel closed successfully: %s", tunnelID)

	return nil
}


func (s *TunnelService) GetAllTunnels() []*model.Tunnel {
	return s.tunnelRepo.GetAll()
}


func (s *TunnelService) GetTunnelByID(tunnelID string) (*model.Tunnel, error) {
	return s.tunnelRepo.GetByID(tunnelID)
}
