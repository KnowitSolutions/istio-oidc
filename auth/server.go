package auth

import (
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"net/http"
	"net/url"
)

var (
	KeycloakURL string
)

type server struct {
	config.Server
	KeyStore
	PolicyStore
	sessionStore
}

func NewServer() *server {
	return &server{
		sessionStore: newSessionStore(),
	}
}

func (srv *server) V2() *ServerV2 {
	return &ServerV2{server: srv}
}

func (srv *server) Start() {
	srv.Validate()
	srv.sessionStore.start()
}

func (srv *server) newRequest(address, cookies string, metadata map[string]string) (*request, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse address", "address", address)
	}

	policy, ok := srv.getAccessPolicy(metadata["accessPolicy"])
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
