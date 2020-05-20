package config

import "time"

type Server struct {
	KeycloakURL string

	SessionCleaning struct {
		Interval    time.Duration
		GracePeriod time.Duration
	}
}

// TODO: Remember to log all errors here as fatal
func (cfg *Server) Validate() {

}
