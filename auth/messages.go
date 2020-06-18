package auth

import (
	"github.com/apex/log"
	"gopkg.in/square/go-jose.v2"
	"istio-keycloak/state"
	"net/http"
	"net/url"
)

const bearerCookie = "bearer"

type request struct {
	url     url.URL
	cookies []*http.Cookie

	accessPolicy string
	oidc         state.OidcCommunicator
	session      *state.Session

	roles  state.Roles
	claims bearerClaims
}

type response struct {
	status  int
	headers map[string]string
}

func (req *request) bearer() string {
	for _, c := range req.cookies {
		if c.Name == bearerCookie {
			return c.Value
		}
	}

	return ""
}

func (req *request) Fields() log.Fields {
	maskQuery := func(query url.Values, field string) {
		if query[field] == nil {
			return
		}

		for i := range query[field] {
			query[field][i] = "-"
		}
	}

	loc := req.url
	query := loc.Query()
	maskQuery(query, "state")
	maskQuery(query, "code")
	loc.RawQuery = query.Encode()

	var bearer string
	tok, err := jose.ParseSigned(req.bearer())
	if err == nil {
		bearer = string(tok.UnsafePayloadWithoutVerification())
	} else {
		bearer = "error"
	}

	return log.Fields{
		"accessPolicy": req.accessPolicy,
		"url":          loc.String(),
		"bearer":       bearer,
		"session":      req.session != nil,
	}
}
