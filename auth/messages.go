package auth

import (
	"github.com/KnowitSolutions/istio-oidc/state/accesspolicy"
	"github.com/KnowitSolutions/istio-oidc/state/session"
	"gopkg.in/square/go-jose.v2"
	"net/http"
	"net/url"
)

const bearerCookie = "bearer"

type request struct {
	url     url.URL
	cookies []*http.Cookie
	claims  bearerClaims

	policy  *accesspolicy.AccessPolicy
	route   *accesspolicy.Route
	session session.Session
}

type response struct {
	status  int
	headers map[string]string
}

func (req *request) location() url.URL {
	loc := req.url
	query := loc.Query()
	for i := range query["state"] {
		query["state"][i] = "-"
	}
	for i := range query["code"] {
		query["state"][i] = "-"
	}
	loc.RawQuery = query.Encode()
	return loc
}

func (req *request) bearer() string {
	for _, c := range req.cookies {
		if c.Name == bearerCookie {
			return c.Value
		}
	}

	return ""
}

func (req *request) rawToken() string {
	tok, err := jose.ParseSigned(req.bearer())
	if err == nil {
		return string(tok.UnsafePayloadWithoutVerification())
	} else {
		return ""
	}
}
