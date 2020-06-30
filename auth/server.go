package auth

import (
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
	"net/http"
	"net/url"
)

type Server struct {
	state.KeyStore
	state.OidcCommunicatorStore
	state.SessionStore
}

func (srv *Server) V2() *ServerV2 {
	return &ServerV2{Server: srv}
}

func (srv *Server) Start() {
	srv.SessionStore.Start()
}

func (srv *Server) newRequest(address, cookies string, metadata map[string]string) (*request, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse address", "address", address)
	}

	oidc, ok := srv.GetOidc(metadata[state.AccessPolicyKey])
	if !ok {
		return nil, errors.New("unknown accessPolicy", "AccessPolicy", metadata[state.AccessPolicyKey])
	}

	roles := &state.Roles{}
	if _, ok = metadata[state.RolesKey]; ok {
		err = roles.Decode(metadata[state.RolesKey])
		if err != nil {
			return nil, errors.Wrap(err, "unable to decode roles")
		}
	}

	req := http.Request{Header: http.Header{}}
	req.Header.Add("Cookie", cookies)

	return &request{
		url:          *parsed,
		cookies:      req.Cookies(),
		accessPolicy: metadata[state.AccessPolicyKey],
		oidc:         oidc,
		roles:        *roles,
	}, nil
}
