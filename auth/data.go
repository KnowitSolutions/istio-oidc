package auth

import (
	"github.com/apex/log"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"istio-keycloak/config"
	"net/http"
	"net/url"
	"time"
)

const bearerCookie = "bearer"

type request struct {
	url     url.URL
	cookies []*http.Cookie

	policy  *accessPolicy
	session *session

	roles  config.Roles
	claims bearerClaims
}

type response struct {
	status  int
	headers map[string]string
}

type session struct {
	refreshToken string
	expiry       time.Time
}

func (req *request) bearer() string {
	for _, c := range req.cookies {
		if c.Name == bearerCookie {
			return c.Value
		}
	}

	return ""
}

func (req *request) oauth2() *oauth2.Config {
	cfg := req.policy.oauth2Config

	// TODO: Check for better solutions
	loc, err := req.url.Parse(req.policy.OIDC.CallbackPath)
	if err != nil {
		log.WithFields(req).WithField("callback", req.policy.OIDC.CallbackPath).
			WithError(err).Fatal("Unable to construct OIDC callback URL")
	}

	cfg.RedirectURL = loc.String()
	cfg.Scopes = []string{oidc.ScopeOpenID}
	return &cfg
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

	policy := ""
	if req.policy != nil {
		policy = req.policy.Name
	}

	var bearer string
	tok, err := jose.ParseSigned(req.bearer())
	if err == nil {
		bearer = string(tok.UnsafePayloadWithoutVerification())
	} else {
		bearer = "error"
	}

	return log.Fields{
		"accessPolicy": policy,
		"url":          loc.String(),
		"bearer":       bearer,
		"session":      req.session != nil,
	}
}
