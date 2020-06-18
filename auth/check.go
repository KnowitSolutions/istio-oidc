package auth

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"golang.org/x/oauth2"
	"istio-keycloak/logging/errors"
	"net/http"
	"strings"
	"time"
)

const subjectHeader = "x-subject"

type stateClaims struct {
	expirableImpl
	Path string `json:"path"`
}

func (srv *server) check(ctx context.Context, req *request) *response {
	if req.policy == nil {
		log.WithFields(req).Error("Invalid accessPolicy")
		return &response{status: http.StatusInternalServerError}
	} else if req.url.Path == req.policy.OIDC.CallbackPath {
		return srv.finishOIDC(ctx, req)
	} else if !srv.isAuthenticated(req) {
		return srv.startOIDC(req)
	} else if req.claims.isExpired() {
		return srv.updateToken(ctx, req)
	} else {
		return srv.authorize(req)
	}
}

func (srv *server) isAuthenticated(req *request) bool {
	token := req.bearer()
	if token == "" {
		return false
	}

	err := parseToken(srv.getKey(), token, &req.claims)
	if err != nil {
		log.WithError(err).Error("Unable to check authentication")
		return false
	}

	var ok bool
	req.session, ok = srv.getSession(token)
	return ok
}

func (srv *server) startOIDC(req *request) *response {
	log.WithFields(req).Info("Starting OIDC")

	claims := &stateClaims{Path: req.url.Path}
	tok, err := makeToken(srv.getKey(), claims, time.Time{})
	if err != nil {
		log.WithError(err).Error("Unable to start OIDC flow")
		return &response{status: http.StatusInternalServerError}
	}

	cfg := req.oauth2()
	loc := cfg.AuthCodeURL(tok)

	headers := map[string]string{"location": loc}
	return &response{status: http.StatusSeeOther, headers: headers}
}

func (srv *server) finishOIDC(ctx context.Context, req *request) *response {
	log.WithFields(req).Info("Finishing OIDC")

	query := req.url.Query()
	if query["state"] == nil || len(query["state"]) != 1 ||
		query["code"] == nil || len(query["code"]) != 1 {
		log.WithFields(req).Warn("Invalid OIDC callback")
		return &response{status: http.StatusBadRequest}
	}

	claims := &stateClaims{}
	err := parseToken(srv.getKey(), query["state"][0], claims)
	if err != nil {
		log.WithError(err).Error("Unable to finnish OIDC flow")
		return &response{status: http.StatusBadRequest}
	}

	cfg := req.oauth2()
	tok, err := cfg.Exchange(ctx, query["code"][0])
	if err != nil {
		err = errors.Wrap(err,"failed authorization code exchange")
		log.WithFields(req).WithFields(log.Fields{
			"clientId": cfg.ClientID,
			"url": cfg.Endpoint.TokenURL,
			"roles": strings.Join(cfg.Scopes, ","),
		}).WithError(err).Error("Unable to finnish OIDC flow")
		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(ctx, req, tok, claims.Path)
}

func (srv *server) updateToken(ctx context.Context, req *request) *response {
	log.WithFields(req).Info("Updating JWT")

	cfg := req.oauth2()
	src := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: req.session.refreshToken})

	tok, err := src.Token()
	if err != nil {
		err = errors.Wrap(err, "failed getting access token")
		log.WithFields(req).WithError(err).Warn("Unable to refresh access token")
		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(ctx, req, tok, "")
}

// TODO: This calls JWKs endpoint twice. That is unnecessary. See if it is possible to merge.
func (srv *server) setToken(ctx context.Context, req *request, token *oauth2.Token, uri string) *response {
	claims, err := makeBearerClaims(ctx, req, token)
	if err != nil {
		log.WithFields(req).WithError(err).Error("Unable to set access token")
		return &response{status: http.StatusInternalServerError}
	}

	tok, err := makeToken(srv.getKey(), claims, token.Expiry)
	if err != nil {
		log.WithFields(req).WithError(err).Error("Unable to set access token")
		return &response{status: http.StatusInternalServerError}
	}

	srv.setSession(tok, token)
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

func (srv *server) authorize(req *request) *response {
	log.WithFields(req).Info("Authorizing")

	found := make(map[string]map[string]bool, len(req.roles))
	tot := 0
	for k1, v := range req.roles {
		found[k1] = make(map[string]bool, len(v))
		for _, k2 := range v {
			found[k1][k2] = false
			tot++
		}
	}

	for k1, v := range req.claims.Roles {
		if _, ok := found[k1]; !ok {
			continue
		}

		for _, k2 := range v {
			if _, ok := found[k1][k2]; !ok {
				continue
			}

			found[k1][k2] = true
		}
	}

	missing := make([]string, 0, tot)
	for k1, v1 := range found {
		for k2, v2 := range v1 {
			if !v2 {
				var str string
				if k1 == "" {
					str = k2
				} else {
					str = fmt.Sprintf("%s/%s", k1, k2)
				}
				missing = append(missing, str)
			}
		}
	}

	if len(missing) == 0 {
		log.WithFields(req).Info("Allowing request")
		headers := map[string]string{subjectHeader: req.claims.Subject}
		return &response{status: http.StatusOK, headers: headers}
	} else {
		log.WithField("missingRoles", strings.Join(missing, ",")).
			WithFields(req).Info("Denying request")
		return &response{status: http.StatusForbidden}
	}
}
