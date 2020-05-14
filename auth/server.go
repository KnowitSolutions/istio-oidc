package auth

import (
	"crypto/sha512"
	"github.com/apex/log"
	"istio-keycloak/config"
)

type server struct {
	config.Server
	Key []byte
	services map[string]*service
	sessions map[[sha512.Size]byte]*session
}

// TODO: Start a sessions cleaner too
func NewServer() *server {
	return &server{
		services: map[string]*service{},
		sessions: map[[sha512.Size]byte]*session{},
	}
}

func (srv *server) V2() *ServerV2 {
	return &ServerV2{server: srv}
}

func (srv *server) V3() *ServerV3 {
	return &ServerV3{server: srv}
}

func (srv *server) AddService(cfg *config.Service) error {
	err := cfg.Validate()
	if err != nil {
		log.WithError(err).Error("Invalid service config") // TODO: Move log to validate
		return err
	}

	svc, err := newService(srv.KeycloakURL, cfg)
	if err != nil {
		return err
	}

	srv.services[cfg.Name] = svc
	return nil
}

