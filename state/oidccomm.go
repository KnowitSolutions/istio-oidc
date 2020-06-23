package state

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/json"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/logging/errors"
	"net/http"
	"net/url"
	"time"
)

var (
	KeycloakURL string
)

type providerMetadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKsURI               string `json:"jwks_uri"`
}

type AccessToken struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

type IdToken struct {
	Subject string `json:"sub"`
}

type OidcCommunicator interface {
	IsCallback(url.URL) bool
	OAuth2(url.URL) *oauth2.Config
	ExtractTokens(context.Context, *oauth2.Token) (*AccessToken, *IdToken, error)
}

type oidcCommunicatorImpl struct {
	cfg      OIDC
	provider providerMetadata
}

func newOIDCCommunicator(ctx context.Context, cfg *AccessPolicy) (OidcCommunicator, error) {
	iss := fmt.Sprintf("%s/auth/realms/%s/", KeycloakURL, cfg.Realm)
	addr := iss + "/.well-known/openid-configuration"
	prov := providerMetadata{}
	err := doJsonRequest(ctx, addr, &prov)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch OIDC provider config", "issuer", iss)
	}

	return &oidcCommunicatorImpl{
		cfg:      cfg.OIDC,
		provider: prov,
	}, nil
}

func (oc *oidcCommunicatorImpl) IsCallback(url url.URL) bool {
	return url.Path == oc.cfg.Callback.Path
}

func (oc *oidcCommunicatorImpl) OAuth2(url url.URL) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     oc.cfg.ClientID,
		ClientSecret: oc.cfg.ClientSecret,
		Endpoint:     oauth2.Endpoint{
			AuthURL:   oc.provider.AuthorizationEndpoint,
			TokenURL:  oc.provider.TokenEndpoint,
		},
		RedirectURL: url.ResolveReference(&oc.cfg.Callback).String(),
		Scopes: []string{"openid"},
	}
}

func (oc *oidcCommunicatorImpl) ExtractTokens(ctx context.Context, tok *oauth2.Token) (at *AccessToken, idt *IdToken, err error) {
	jwks := &jose.JSONWebKeySet{}
	err = doJsonRequest(ctx, oc.provider.JWKsURI, jwks)
	if err != nil {
		err = errors.Wrap(err, "unable to retrieve JWKs")
		return
	}

	at = &AccessToken{}
	err = claims(tok.AccessToken, oc.provider.Issuer, oc.cfg.ClientID, jwks, at)
	if err != nil {
		err = errors.Wrap(err, "unable to get access token claims")
		return
	}

	idt = &IdToken{}
	err = claims(tok.Extra("id_token").(string), oc.provider.Issuer, oc.cfg.ClientID, jwks, idt)
	if err != nil {
		err = errors.Wrap(err, "unable to get ID token claims")
		return
	}

	return
}

func doJsonRequest(ctx context.Context, url string, data interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return errors.Wrap(err, "failed preparing request", "url", url)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "communication error", "url", url)
	}
	defer func() { _ = res.Body.Close() }()

	err = json.NewDecoder(res.Body).Decode(data)
	if err != nil {
		return errors.Wrap(err, "failed decoding JSON", "url", url)
	}

	return nil
}

func claims(tok, iss, aud string, jwks *jose.JSONWebKeySet, claims interface{}) error {
	parsed, err := jwt.ParseSigned(tok)
	if err != nil {
		return errors.Wrap(err, "failed paring token", "token", tok)
	}

	def := &jwt.Claims{}
	err = parsed.Claims(jwks, def)
	if err != nil {
		return errors.Wrap(err, "failed deserializing default claims", "token", tok)
	}

	exp := jwt.Expected{
		Issuer:   iss,
		Audience: jwt.Audience{aud},
		Time:     time.Now(),
	}
	err = def.ValidateWithLeeway(exp, 0)
	if err != nil {
		return errors.Wrap(err, "failed validating token", "token", tok)
	}

	err = parsed.Claims(jwks, claims)
	if err != nil {
		return errors.Wrap(err, "failed deserializing custom claims", "token", tok)
	}

	return nil
}
