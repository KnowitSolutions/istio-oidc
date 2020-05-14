package config

import "time"

type Server struct {
	KeycloakURL string
	TokenDuration time.Duration
}

func (cfg *Server) Validate() error {
	return nil // TODO
}
