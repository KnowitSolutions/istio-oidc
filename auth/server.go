package auth

import (
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"github.com/KnowitSolutions/istio-oidc/replication"
	"github.com/KnowitSolutions/istio-oidc/state/accesspolicy"
	"github.com/KnowitSolutions/istio-oidc/state/session"
	"net/http"
	"net/url"
)

type Server struct {
	replication.Client
	AccessPolicies accesspolicy.Store
	Sessions       session.Store
}

func (srv *Server) V2() *ServerV2 {
	return &ServerV2{Server: srv}
}

func (srv *Server) newRequest(address, cookies, fetchMode string, metadata map[string]string) (*request, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse address", "address", address)
	}

	ap := srv.AccessPolicies.Get(metadata[accesspolicy.NameKey])
	if ap == nil {
		return nil, errors.New("unknown AccessPolicy", "AccessPolicy", metadata[accesspolicy.NameKey])
	}

	route, ok := ap.Routes[metadata[accesspolicy.RouteKey]]
	if metadata[accesspolicy.RouteKey] == "" {
		route = ap.Default
	} else if !ok {
		return nil, errors.New("unknown route", "AccessPolicy", metadata[accesspolicy.NameKey])
	}

	req := http.Request{Header: http.Header{}}
	req.Header.Add("Cookie", cookies)

	return &request{
		url:       *parsed,
		cookies:   req.Cookies(),
		fetchMode: fetchMode,

		policy: ap,
		route:  &route,
	}, nil
}
