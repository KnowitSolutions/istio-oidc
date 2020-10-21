package auth

import (
	"context"
	"crypto/sha512"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"github.com/KnowitSolutions/istio-oidc/state/session"
	"golang.org/x/oauth2"
	"net/http"
	"strings"
	"time"
)

type stateClaims struct {
	claims
	Path string `json:"path"`
}

type bearerClaims struct {
	claims
	Roles map[string][]string `json:"rol"`
}

func (srv *Server) check(ctx context.Context, req *request) *response {
	var res *response
	loc := req.location()
	ctx = log.WithValues(ctx, "url", loc.String(), "bearer", req.rawToken(), "AccessPolicy", req.policy.Name)

	if req.policy.Oidc.IsCallback(req.url) {
		reqCallbackCount.WithLabelValues(req.policy.Name).Inc()
		res = srv.finishOidc(ctx, req)
	} else if !srv.isAuthenticated(ctx, req) {
		reqUnauthdCount.WithLabelValues(req.policy.Name).Inc()
		res = srv.startOidc(ctx, req)
	} else if req.claims.isExpired() {
		reqExpiredCount.WithLabelValues(req.policy.Name).Inc()
		res = srv.updateToken(ctx, req)
	} else {
		reqAuthdCount.WithLabelValues(req.policy.Name).Inc()
		res = srv.authorize(ctx, req)
	}

	switch res.status {
	case http.StatusOK:
		resAllowedCount.WithLabelValues(req.policy.Name).Inc()
	case http.StatusSeeOther:
		fallthrough
	case http.StatusTemporaryRedirect:
		resRedirCount.WithLabelValues(req.policy.Name).Inc()
	case http.StatusBadRequest:
		resBadReqCount.WithLabelValues(req.policy.Name).Inc()
	case http.StatusForbidden:
		resDeniedCount.WithLabelValues(req.policy.Name).Inc()
	case http.StatusInternalServerError:
		resErrorCount.WithLabelValues(req.policy.Name).Inc()
	default:
		resOtherCount.WithLabelValues(req.policy.Name).Inc()
	}

	return res
}

func (srv *Server) isAuthenticated(ctx context.Context, req *request) bool {
	token := req.bearer()
	if token == "" {
		return false
	}

	err := parseToken(req.policy.Oidc.TokenSecret, token, &req.claims)
	if err != nil {
		log.Error(ctx, err, "Unable to check authentication")
		return false
	}

	hash := sha512.Sum512([]byte(token))
	id := string(hash[:])

	var ok bool
	req.session, ok = srv.Sessions.Get(id)
	return ok
}

func (srv *Server) startOidc(ctx context.Context, req *request) *response {
	switch req.fetchMode {
	case "":
	case "navigate":
	case "nested-navigate":
	default:
		log.Info(ctx, nil, "Skipping OIDC for non-navigation request")
		return &response{status: http.StatusForbidden}
	}

	log.Info(ctx, nil, "Starting OIDC")

	claims := &stateClaims{Path: req.url.Path}
	tok, err := makeToken(req.policy.Oidc.TokenSecret, claims, time.Time{})
	if err != nil {
		log.Error(ctx, err, "Unable to start OIDC flow")
		return &response{status: http.StatusInternalServerError}
	}

	cfg := req.policy.Oidc.OAuth2(req.url)
	loc := cfg.AuthCodeURL(tok)

	headers := map[string]string{"location": loc}
	return &response{status: http.StatusSeeOther, headers: headers}
}

func (srv *Server) finishOidc(ctx context.Context, req *request) *response {
	log.Info(ctx, nil, "Finishing OIDC")

	query := req.url.Query()
	if query["state"] == nil || len(query["state"]) != 1 ||
		query["code"] == nil || len(query["code"]) != 1 {
		log.Error(ctx, nil, "Invalid OIDC callback")
		return &response{status: http.StatusBadRequest}
	}

	claims := &stateClaims{}
	err := parseToken(req.policy.Oidc.TokenSecret, query["state"][0], claims)
	if err != nil {
		log.Error(ctx, nil, "Unable to finnish OIDC flow")
		return &response{status: http.StatusBadRequest}
	}

	cfg := req.policy.Oidc.OAuth2(req.url)
	tok, err := cfg.Exchange(ctx, query["code"][0])
	if err != nil {
		err = errors.Wrap(
			err, "failed authorization code exchange",
			"clientId", cfg.ClientID,
			"url", cfg.Endpoint.TokenURL,
			"roles", strings.Join(cfg.Scopes, ","))
		log.Error(ctx, err, "Unable to finnish OIDC flow")
		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(ctx, req, tok, claims.Path)
}

func (srv *Server) updateToken(ctx context.Context, req *request) *response {
	log.Info(ctx, nil, "Updating JWT")

	cfg := req.policy.Oidc.OAuth2(req.url)
	src := cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: req.session.RefreshToken})

	tok, err := src.Token()
	if err != nil {
		err = errors.Wrap(err, "failed getting access token")
		log.Error(ctx, err, "Unable to refresh access token")
		return &response{status: http.StatusForbidden}
	}

	return srv.setToken(ctx, req, tok, "")
}

func (srv *Server) setToken(ctx context.Context, req *request, token *oauth2.Token, uri string) *response {
	data, err := req.policy.Oidc.Provider.TokenData(ctx, *token)
	if err != nil {
		log.Error(ctx, err, "Unable to set access token")
		return &response{status: http.StatusInternalServerError}
	}

	claims := bearerClaims{}
	claims.Subject = data.Subject
	claims.Roles = data.Roles

	tok, err := makeToken(req.policy.Oidc.TokenSecret, claims, token.Expiry)
	if err != nil {
		log.Error(ctx, err, "Unable to set access token")
		return &response{status: http.StatusInternalServerError}
	}

	hash := sha512.Sum512([]byte(tok))
	id := string(hash[:])
	sess := session.Stamped{
		Session: session.Session{
			Id:           id,
			RefreshToken: token.RefreshToken,
			Expiry:       token.Expiry,
		},
	}
	sess, _ = srv.Sessions.Set(sess)
	srv.Client.SetSession(sess)

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

func (srv *Server) authorize(ctx context.Context, req *request) *response {
	log.Info(ctx, nil, "Authorizing")

	if !hasRoles(req.route.Roles, req.claims.Roles) {
		log.Info(ctx, nil, "Denying request")
		return &response{status: http.StatusForbidden}
	} else {
		log.Info(ctx, nil, "Allowing request")
	}

	headers := make(map[string]string, len(req.route.Headers))
	for _, header := range req.route.Headers {
		if hasRoles(header.Roles, req.claims.Roles) {
			headers[header.Name] = header.Value
		}
	}

	return &response{status: http.StatusOK, headers: headers}
}
