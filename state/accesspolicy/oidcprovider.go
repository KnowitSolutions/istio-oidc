package accesspolicy

import (
	"context"
	"encoding/json"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"net/http"
	"time"
)

type oidcProvider struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKsURI               string `json:"jwks_uri"`
}

func (prov *oidcProvider) updateOidcProvider(ctx context.Context, realm string) error {
	iss := config.Keycloak.Url + "/auth/realms/" + realm
	addr := iss + "/.well-known/openid-configuration"

	err := doJsonRequest(ctx, addr, prov)
	if err != nil {
		err = errors.Wrap(err, "unable to fetch OIDC provider config", "issuer", iss)
		return err
	}

	return nil
}

type tokenData struct {
	Subject string
	Roles   Roles
}

type accessToken struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

type idToken struct {
	Subject string `json:"sub"`
}

func (prov oidcProvider) ExtractTokenData(ctx context.Context, tok *oauth2.Token) (tokenData, error) {
	jwks := &jose.JSONWebKeySet{}
	err := doJsonRequest(ctx, prov.JWKsURI, jwks)
	if err != nil {
		return tokenData{}, errors.Wrap(err, "unable to retrieve JWKs")
	}

	at := accessToken{}
	err = claims(tok.AccessToken, jwks, &at)
	if err != nil {
		return tokenData{}, errors.Wrap(err, "unable to get access token claims")
	}

	idt := idToken{}
	err = claims(tok.Extra("id_token").(string), jwks, &idt)
	if err != nil {
		return tokenData{}, errors.Wrap(err, "unable to get ID token claims")
	}

	count := len(at.RealmAccess.Roles)
	for _, v := range at.ResourceAccess {
		count += len(v.Roles)
	}

	roles := make([]string, 0, count)
	for _, v := range at.RealmAccess.Roles {
		roles = append(roles, v)
	}
	for k, v := range at.ResourceAccess {
		for _, v := range v.Roles {
			roles = append(roles, k+"/"+v)
		}
	}

	return tokenData{
		Subject: idt.Subject,
		Roles: roles,
	}, nil
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

func claims(tok string, jwks *jose.JSONWebKeySet, claims interface{}) error {
	parsed, err := jwt.ParseSigned(tok)
	if err != nil {
		return errors.Wrap(err, "failed parsing token", "token", tok)
	}

	def := &jwt.Claims{}
	err = parsed.Claims(jwks, def)
	if err != nil {
		return errors.Wrap(err, "failed deserializing default claims", "token", tok)
	}

	exp := jwt.Expected{Time: time.Now()}
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
