package auth

import (
	"istio-keycloak/log/errors"
	"istio-keycloak/state"
	"istio-keycloak/state/accesspolicy"
	"net/http"
	"net/url"
)

type Server struct {
	state.KeyStore
	state.AccessPolicyStore
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

	ap := srv.GetAccessPolicy(metadata[accesspolicy.NameKey])
	if ap == nil {
		return nil, errors.New("unknown " + accesspolicy.NameKey, "AccessPolicy", metadata[accesspolicy.NameKey])
	}

	route, ok := ap.Routes[metadata[accesspolicy.RouteKey]]
	if metadata[accesspolicy.RouteKey] == "" {
		route = ap.Default
	} else if !ok {
		return nil, errors.New("unknown " + accesspolicy.RouteKey, "AccessPolicy", metadata[accesspolicy.NameKey])
	}

	req := http.Request{Header: http.Header{}}
	req.Header.Add("Cookie", cookies)

	return &request{
		url:     *parsed,
		cookies: req.Cookies(),
		policy:  ap,
		route:   &route,
	}, nil
}
