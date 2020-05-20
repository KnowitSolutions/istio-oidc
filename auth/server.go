package auth

import (
	"crypto/sha512"
	"github.com/apex/log"
	"istio-keycloak/config"
	"sync"
	"time"
)

type server struct {
	config.Server
	Key        []byte
	services   map[string]*service
	servicesMu sync.RWMutex
	sessions   map[[sha512.Size]byte]*session
	sessionsMu sync.RWMutex
}

func NewServer() *server {
	srv := &server{
		services: map[string]*service{},
		sessions: map[[sha512.Size]byte]*session{},
	}
	go srv.sessionCleaner()
	return srv
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
		return err
	}

	svc, err := newService(srv.KeycloakURL, cfg)
	if err != nil {
		return err
	}

	srv.servicesMu.Lock()
	srv.services[cfg.Name] = svc
	srv.servicesMu.Unlock()
	return nil
}

func (srv *server) sessionCleaner() {
	tick := time.NewTicker(srv.SessionCleaning.Interval)

	for {
		<-tick.C

		start := time.Now()
		max := start.Add(-srv.SessionCleaning.GracePeriod)
		tot := 0

		log.WithField("max", max.Format(time.RFC3339)).
			Info("Cleaning sessions")

		srv.sessionsMu.RLock()
		for k, v := range srv.sessions {
			if v.expiry.Before(max) {
				srv.sessionsMu.RUnlock()
				srv.sessionsMu.Lock()

				delete(srv.sessions, k)
				tot++

				srv.sessionsMu.Unlock()
				srv.sessionsMu.RLock()
			}
		}
		srv.sessionsMu.RUnlock()

		stop := time.Now()
		dur := stop.Sub(start)
		log.WithFields(log.Fields{"duration": dur, "total": tot}).
			Info("Done cleaning sessions")
	}
}
