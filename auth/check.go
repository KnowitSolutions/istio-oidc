package auth

import (
	"context"
	"github.com/apex/log"
	"golang.org/x/oauth2"
	"istio-keycloak/logging/errors"
	"net/http"
	"strings"
	"time"
)

const subjectHeader = "x-subject"

type stateClaims struct {
	expirable
	Path string `json:"path"`
}

func (srv *Server) check(ctx context.Context, req *request) *response {
	var res *response

	if req.accessPolicyHelper.IsCallback(req.url) {
		reqCallbackCount.WithLabelValues(req.accessPolicy).Inc()
		res = srv.finishOidc(ctx, req)
	} else if !srv.isAuthenticated(req) {
		reqUnauthdCount.WithLabelValues(req.accessPolicy).Inc()
		res = srv.startOidc(req)
	} else if req.claims.isExpired() {
		reqExpiredCount.WithLabelValues(req.accessPolicy).Inc()
		res = srv.updateToken(ctx, req)
	} else {
		reqAuthdCount.WithLabelValues(req.accessPolicy).Inc()
		res = srv.authorize(req)
	}

	switch res.status {
	case http.StatusOK:
		resAllowedCount.WithLabelValues(req.accessPolicy).Inc()
	case http.StatusSeeOther:
		fallthrough
	case http.StatusTemporaryRedirect:
		resRedirCount.WithLabelValues(req.accessPolicy).Inc()
	case http.StatusBadRequest:
		resBadReqCount.WithLabelValues(req.accessPolicy).Inc()
	case http.StatusForbidden:
		resDeniedCount.WithLabelValues(req.accessPolicy).Inc()
	case http.StatusInternalServerError:
		resErrorCount.WithLabelValues(req.accessPolicy).Inc()
	default:
		resOtherCount.WithLabelValues(req.accessPolicy).Inc()
	}

	return res
}

func (srv *Server) isAuthenticated(req *request) bool {
	token := req.bearer()
	if token == "" {
		return false
	}

	err := parseToken(srv.GetKey(), token, &req.claims)
	if err != nil {
		log.WithError(err).Error("Unable to check authentication")
		return false
	}

	var ok bool
	req.session, ok = srv.GetSession(token)
	return ok
}

func (srv *Server) startOidc(req *request) *response {
	log.WithFields(req).Info("Starting OIDC")

	claims := &stateClaims{Path: req.url.Path}
	tok, err := makeToken(srv.GetKey(), claims, time.Time{})
	if err != nil {
		log.WithError(err).Error("Unable to start OIDC flow")
		return &response{status: http.StatusInternalServerError}
	}

	cfg := req.accessPolicyHelper.OAuth2(req.url)
	loc := cfg.AuthCodeURL(tok)

	headers := map[string]string{"location": loc}
	return &response{status: http.StatusSeeOther, headers: headers}
}

func (srv *Server) finishOidc(ctx context.Context, req *request) *response {
	log.WithFields(req).Info("Finishing OIDC")

	query := req.url.Query()
	if query["state"] == nil || len(query["state"]) != 1 ||
		query["code"] == nil || len(query["code"]) != 1 {
		log.WithFields(req).Error("Invalid OIDC callback")
		return &response{status: http.StatusBadRequest}
	}

	claims := &stateClaims{}
	err := parseToken(srv.GetKey(), query["state"][0], claims)
	if err != nil {
		log.WithError(err).Error("Unable to finnish OIDC flow")
		return &response{status: http.StatusBadRequest}
	}

	cfg := req.accessPolicyHelper.OAuth2(req.url)
	tok, err := cfg.Exchange(ctx, query["code"][0])
	if err != nil {
		err = errors.Wrap(err, "failed authorization code exchange")
		log.WithFields(req).WithFields(log.Fields{
			"clientId": cfg.ClientID,
			"url":      cfg.Endpoint.TokenURL,
			"roles":    strings.Join(cfg.Scopes, ","),
		}).WithError(err).Error("Unable to finnish OIDC flow")
		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(ctx, req, tok, claims.Path)
}

func (srv *Server) updateToken(ctx context.Context, req *request) *response {
	log.WithFields(req).Info("Updating JWT")

	cfg := req.accessPolicyHelper.OAuth2(req.url)
	src := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: req.session.RefreshToken})

	tok, err := src.Token()
	if err != nil {
		err = errors.Wrap(err, "failed getting access token")
		log.WithFields(req).WithError(err).Error("Unable to refresh access token")
		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(ctx, req, tok, "")
}

func (srv *Server) setToken(ctx context.Context, req *request, token *oauth2.Token, uri string) *response {
	claims, err := makeBearerClaims(ctx, req, token)
	if err != nil {
		log.WithFields(req).WithError(err).Error("Unable to set access token")
		return &response{status: http.StatusInternalServerError}
	}

	tok, err := makeToken(srv.GetKey(), claims, token.Expiry)
	if err != nil {
		log.WithFields(req).WithError(err).Error("Unable to set access token")
		return &response{status: http.StatusInternalServerError}
	}

	sess, err := req.accessPolicyHelper.CreateSession(token)
	if err != nil {
		log.WithFields(req).WithError(err).Error("Unable to set access token")
		return &response{status: http.StatusInternalServerError}
	}

	srv.SetSession(tok, sess)
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

func (srv *Server) authorize(req *request) *response {
	log.WithFields(req).Info("Authorizing")

	found := make(map[string]bool, len(req.roles))
	for _, k := range req.roles {
		found[k] = false
	}
	for _, k := range req.claims.Roles {
		if _, ok := found[k]; ok {
			found[k] = true
		}
	}

	missing := make([]string, 0, len(found))
	for k, v := range found {
		if !v {
			missing = append(missing, k)
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
