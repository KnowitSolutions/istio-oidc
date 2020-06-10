package auth

import (
	"context"
	"crypto/sha512"
	"github.com/apex/log"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	KeycloakURL string
	SessionCleaningInterval time.Duration
	SessionCleaningGracePeriod time.Duration
)

type server struct {
	config.Server
	Key        []byte
	policies   map[string]*accessPolicy
	policiesMu sync.RWMutex
	sessions   map[[sha512.Size]byte]*session
	sessionsMu sync.RWMutex
}

func NewServer() *server {
	return &server{
		policies: map[string]*accessPolicy{},
		sessions: map[[sha512.Size]byte]*session{},
	}
}

func (srv *server) V2() *ServerV2 {
	return &ServerV2{server: srv}
}

func (srv *server) AddAccessPolicy(ctx context.Context, cfg *config.AccessPolicy) error {
	pol, err := newAccessPolicy(ctx, KeycloakURL, cfg)
	if err != nil {
		return errors.Wrap(err, "unable to add accessPolicy")
	}

	srv.policiesMu.Lock()
	srv.policies[cfg.Name] = pol
	srv.policiesMu.Unlock()
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

	srv.policiesMu.RLock()
	policy, ok := srv.policies[metadata["accessPolicy"]]
	srv.policiesMu.RUnlock()
	if !ok {
		return nil, errors.New("unknown accessPolicy", "accessPolicy", metadata["accessPolicy"])
	}

	roles := &config.Roles{}
	if _, ok = metadata["roles"]; ok {
		err = roles.Decode(metadata["roles"])
		if err != nil {
			return nil, errors.Wrap(err, "unable to decode roles")
		}
	}

	req := http.Request{Header: http.Header{}}
	req.Header.Add("Cookie", cookies)

	return &request{
		url:     *parsed,
		cookies: req.Cookies(),
		policy:  policy,
		roles:   *roles,
	}, nil
}

func (srv *server) sessionCleaner() {
	tick := time.NewTicker(SessionCleaningInterval)

	for {
		<-tick.C

		start := time.Now()
		max := start.Add(-SessionCleaningGracePeriod)
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
