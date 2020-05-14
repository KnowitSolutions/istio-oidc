package auth

import (
	"context"
	"crypto/sha512"
	"github.com/apex/log"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
	"time"
)

type stateClaims struct {
	expirableImpl
	Path string `json:"path"`
}

func (srv *server) check(req *request, roles []string) *response {
	if req.service == nil {
		log.WithFields(req).Warn("Invalid service")
		return &response{status: http.StatusBadRequest}
	} else if req.url.Path == req.service.OIDC.CallbackPath {
		return srv.finishOIDC(req)
	} else if !srv.isAuthenticated(req) {
		return srv.startOIDC(req)
	} else if req.claims.isExpired() {
		return srv.updateToken(req)
	} else {
		return srv.authorize(req, roles)
	}
}

func (srv *server) isAuthenticated(req *request) bool {
	token := req.bearer()
	if token == "" {
		return false
	}

	err := parseToken(srv.Key, token, &req.claims)
	if err != nil {
		return false
	}

	hash := sha512.Sum512([]byte(token))
	var ok bool
	req.session, ok = srv.sessions[hash]
	return ok
}

func (srv *server) startOIDC(req *request) *response {
	log.WithFields(req).Info("Starting OIDC")

	claims := &stateClaims{Path: req.url.Path}
	tok := makeToken(srv.Key, claims, time.Time{})

	cfg := req.oauth2()
	loc := cfg.AuthCodeURL(tok)

	headers := map[string]string{"location": loc}
	return &response{status: http.StatusSeeOther, headers: headers}
}

func (srv *server) finishOIDC(req *request) *response {
	log.WithFields(req).Info("Finishing OIDC")

	query := req.url.Query()
	if query["state"] == nil || len(query["state"]) != 1 ||
		query["code"] == nil || len(query["code"]) != 1 {
		log.WithFields(req).Warn("Invalid OIDC callback")
		return &response{status: http.StatusBadRequest}
	}

	claims := &stateClaims{}
	err := parseToken(srv.Key, query["state"][0], claims)
	if err != nil {
		return &response{status: http.StatusBadRequest}
	}

	cfg := req.oauth2()
	tok, err := cfg.Exchange(context.TODO(), query["code"][0])
	if err != nil {
		log.WithFields(log.Fields{
			"clientId": cfg.ClientID,
			"url":      cfg.Endpoint.TokenURL,
			"roles":    strings.Join(cfg.Scopes, ","),
		}).WithError(err).Error("Unable to exchange authorization code")

		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(req, tok, claims.Path)
}

func (srv *server) updateToken(req *request) *response {
	log.WithFields(req).Info("Updating JWT")

	cfg := req.oauth2()
	src := cfg.TokenSource(context.TODO(), &oauth2.Token{RefreshToken: req.session.refreshToken})

	tok, err := src.Token()
	if err != nil {
		log.WithFields(req).WithError(err).Warn("Unable to refresh access token")
		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(req, tok, "")
}

// TODO: This calls JWKs endpoint twice. That is unnecessary. See if it is possible to merge.
func (srv *server) setToken(req *request, token *oauth2.Token, uri string) *response {
	claims, err := makeBearerClaims(req, token)
	if err != nil {
		return &response{status: http.StatusInternalServerError}
	}

	tok := makeToken(srv.Key, claims, token.Expiry)
	hash := sha512.Sum512([]byte(tok))
	srv.sessions[hash] = &session{
		refreshToken: token.RefreshToken,
		expiry:       token.Expiry,
	}
	// TODO: Gossip

	cookie := http.Cookie{
		Name:     bearerCookie,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
	}
	res := &response{headers: map[string]string{"set-cookie": cookie.String()}}

	if uri == "" {
		res.status = http.StatusTemporaryRedirect
		res.headers["location"] = req.url.RequestURI()
	} else {
		res.status = http.StatusSeeOther
		res.headers["location"] = uri
	}

	return res
}

func (srv *server) authorize(req *request, roles []string) *response {
	log.WithFields(req).Info("Authorizing")

	roles2 := make(map[string]bool, len(roles))
	for _, k := range roles {
		roles2[k] = false
	}
	for _, k := range req.claims.ResourceAccess[req.service.Name] {
		roles2[k] = true
	}

	missing := make([]string, 0, len(roles))
	for k, v := range roles2 {
		if !v {
			missing = append(missing, k)
		}
	}

	if len(missing) == 0 {
		log.WithFields(req).Info("Allowing request")
		return &response{status: http.StatusOK}
	} else {
		log.WithField("missingRoles", strings.Join(missing, ",")).
			WithFields(req).Info("Denying request")
		return &response{status: http.StatusForbidden}
	}
}
