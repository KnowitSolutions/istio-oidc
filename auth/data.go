package auth

import (
	"context"
	"github.com/apex/log"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"net/http"
	"net/url"
	"time"
)

const bearerCookie = "bearer"

type request struct {
	ctx     context.Context
	url     url.URL
	cookies []*http.Cookie

	service *service
	session *session

	roles  map[string][]string
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
	cfg := req.service.oauth2Config

	loc, err := req.url.Parse(req.service.OIDC.CallbackPath)
	if err != nil {
		log.WithFields(req).WithField("callback", req.service.OIDC.CallbackPath).
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

	var bearer string
	tok, err := jose.ParseSigned(req.bearer())
	if err == nil {
		bearer = string(tok.UnsafePayloadWithoutVerification())
	} else {
		bearer = "error"
	}

	return log.Fields{
		"service": req.service.Name,
		"url":     loc.String(),
		"bearer":  bearer,
		"session": req.session != nil,
	}
}
