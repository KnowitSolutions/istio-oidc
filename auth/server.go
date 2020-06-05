package auth

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/gob"
	"github.com/apex/log"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"net/http"
	"net/url"
	"strings"
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
	return &server{
		services: map[string]*service{},
		sessions: map[[sha512.Size]byte]*session{},
	}
}

func (srv *server) V2() *ServerV2 {
	return &ServerV2{server: srv}
}

func (srv *server) AddService(ctx context.Context, cfg *config.Service) error {
	err := cfg.Validate()
	if err != nil {
		return errors.Wrap(err, "unable to add service")
	}

	svc, err := newService(ctx, srv.KeycloakURL, cfg)
	if err != nil {
		return errors.Wrap(err, "unable to add service")
	}

	srv.servicesMu.Lock()
	srv.services[cfg.Name] = svc
	srv.servicesMu.Unlock()
	return nil
}

func (srv *server) Start() {
	srv.Validate()
	go srv.sessionCleaner()
}

func (srv *server) newRequest(address, cookies string, metadata map[string]string) (*request, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse address", "address", address)
	}

	srv.servicesMu.RLock()
	service, ok := srv.services[metadata["service"]]
	srv.servicesMu.RUnlock()
	if !ok {
		return nil, errors.New("unknown service", "service", metadata["service"])
	}

	roles := &config.Roles{}
	if _, ok = metadata["roles"]; ok {
		buf := strings.NewReader(metadata["roles"])
		dec := gob.NewDecoder(base64.NewDecoder(base64.StdEncoding, buf))

		err = dec.Decode(roles)
		if err != nil {
			return nil, errors.Wrap(err, "unable to decode roles")
		}
	}

	req := http.Request{Header: http.Header{}}
	req.Header.Add("Cookie", cookies)

	return &request{
		url:     *parsed,
		cookies: req.Cookies(),
		service: service,
		roles:   *roles,
	}, nil
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
