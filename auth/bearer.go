package auth

import (
	"context"
	"encoding/json"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"net/http"
)

type providerClaims struct {
	JWKsURI string `json:"jwks_uri"`
}

type accessClaims struct {
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

type bearerClaims struct {
	expirableImpl
	jwt.Claims
	Roles map[string][]string `json:"rol"`
}

func makeBearerClaims(ctx context.Context, req *request, tok *oauth2.Token) (*bearerClaims, error) {
	accTok, err := jwt.ParseSigned(tok.AccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse access token", "token", tok.AccessToken)
	}

	provClaims := &providerClaims{}
	err = req.service.oidcProvider.Claims(provClaims)
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve provider claims")
	}

	r, err := http.NewRequestWithContext(ctx, "GET", provClaims.JWKsURI, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make request object", "url", provClaims.JWKsURI)
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve JWKs", "url", provClaims.JWKsURI)
	}
	defer func() { _ = res.Body.Close() }()

	jwks := &jose.JSONWebKeySet{}
	err = json.NewDecoder(res.Body).Decode(jwks)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse JWKs", "url", provClaims.JWKsURI)
	}

	accClaims := &accessClaims{}
	err = accTok.Claims(jwks, accClaims)
	if err != nil {
		return nil, errors.Wrap(err, "unable to deserialize access token", "token", tok)
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("unable to extract ID token")
	}

	idToken, err := req.service.oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, errors.Wrap(err, "got invalid ID token", "token", rawIDToken)
	}

	claims := &bearerClaims{}
	claims.Subject = idToken.Subject
	claims.Roles = make(map[string][]string, len(accClaims.ResourceAccess)+1)
	claims.Roles[config.GlobalRoleKey] = accClaims.RealmAccess.Roles
	for k, v := range accClaims.ResourceAccess {
		claims.Roles[k] = v.Roles
	}

	return claims, nil
}
