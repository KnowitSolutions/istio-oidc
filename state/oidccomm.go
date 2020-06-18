package state

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/json"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/logging/errors"
	"net/http"
	"net/url"
)

var (
	KeycloakURL string
)

type AccessToken struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

type OidcCommunicator interface {
	IsCallback(url.URL) bool
	OAuth2(url.URL) *oauth2.Config
	ExtractTokens(context.Context, *oauth2.Token) (*AccessToken, *oidc.IDToken, error)
}

type oidcCommunicatorImpl struct {
	callback url.URL
	cfg      oauth2.Config
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

func newOIDCCommunicator(ctx context.Context, cfg *AccessPolicy) (OidcCommunicator, error) {
	callback, err := url.Parse(cfg.OIDC.CallbackPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse callback URL", "url", cfg.OIDC.CallbackPath)
	}

	iss := fmt.Sprintf("%s/auth/realms/%s", KeycloakURL, cfg.Realm)
	prov, err := oidc.NewProvider(ctx, iss)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch OIDC provider config", "issuer", iss)
	}

	oauth2cfg := oauth2.Config{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Endpoint:     prov.Endpoint(),
	}
	verifier := prov.Verifier(&oidc.Config{ClientID: cfg.OIDC.ClientID})

	return &oidcCommunicatorImpl{
		callback: *callback,
		cfg:      oauth2cfg,
		provider: prov,
		verifier: verifier,
	}, nil
}

func (oc *oidcCommunicatorImpl) IsCallback(url url.URL) bool {
	return url.Path == oc.callback.Path
}

func (oc *oidcCommunicatorImpl) OAuth2(url url.URL) *oauth2.Config {
	cfg := oc.cfg
	cfg.RedirectURL = url.ResolveReference(&oc.callback).String()
	cfg.Scopes = []string{oidc.ScopeOpenID}
	return &cfg
}

// TODO: This calls JWKs endpoint twice. That is unnecessary. See if it is possible to merge.
func (oc *oidcCommunicatorImpl) ExtractTokens(ctx context.Context, tok *oauth2.Token) (at *AccessToken, idt *oidc.IDToken, err error) {
	accTok, err := jwt.ParseSigned(tok.AccessToken)
	if err != nil {
		err = errors.Wrap(err, "unable to parse access token", "token", tok.AccessToken)
		return
	}

	provClaims := &struct {
		JWKsURI string `json:"jwks_uri"`
	}{}
	err = oc.provider.Claims(provClaims)
	if err != nil {
		err = errors.Wrap(err, "unable to retrieve provider claims")
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", provClaims.JWKsURI, nil)
	if err != nil {
		err = errors.Wrap(err, "unable to make request object", "url", provClaims.JWKsURI)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		err = errors.Wrap(err, "unable to retrieve JWKs", "url", provClaims.JWKsURI)
		return
	}
	defer func() { _ = res.Body.Close() }()

	jwks := &jose.JSONWebKeySet{}
	err = json.NewDecoder(res.Body).Decode(jwks)
	if err != nil {
		err = errors.Wrap(err, "unable to parse JWKs", "url", provClaims.JWKsURI)
		return
	}

	err = accTok.Claims(jwks, at)
	if err != nil {
		err = errors.Wrap(err, "unable to deserialize access token", "token", tok)
		return
	}

	rawIdt, ok := tok.Extra("id_token").(string)
	if !ok {
		err = errors.New("unable to extract ID token")
		return
	}

	idt, err = oc.verifier.Verify(ctx, rawIdt)
	if err != nil {
		err = errors.Wrap(err, "got invalid ID token", "token", rawIdt)
		return
	}

	return
}
