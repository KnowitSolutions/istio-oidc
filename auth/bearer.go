package auth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/apex/log"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/config"
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
		log.WithField("token", tok.AccessToken).WithError(err).
			Error("Unable to parse access token")
		return nil, err
	}

	provClaims := &providerClaims{}
	err = req.service.oidcProvider.Claims(provClaims)
	if err != nil {
		log.WithError(err).Fatal("Unable to retrieve provider claims")
	}

	r, err := http.NewRequestWithContext(ctx, "GET", provClaims.JWKsURI, nil)
	if err != nil {
		log.WithField("url", provClaims.JWKsURI).
			WithError(err).Fatal("Unable to make request object")
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		log.WithField("url", provClaims.JWKsURI).
			WithError(err).Error("Unable to retrieve JWKs")
		return nil, err
	}

	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.WithError(err).Fatal("Unable to clean up response body")
		}
	}()

	jwks := &jose.JSONWebKeySet{}
	err = json.NewDecoder(res.Body).Decode(jwks)
	if err != nil {
		log.WithField("url", provClaims.JWKsURI).
			WithError(err).Error("Unable to parse JWKs")
		return nil, err
	}

	accClaims := &accessClaims{}
	err = accTok.Claims(jwks, accClaims)
	if err != nil {
		log.WithField("token", tok).WithError(err).
			Error("Unable to deserialize access token")
		return nil, err
	}

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		log.Error("Unable to extract ID token")
		return nil, errors.New("unable to extract ID token")
	}

	idToken, err := req.service.oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.WithField("token", rawIDToken).WithError(err).
			Error("Got invalid ID token")
		return nil, err
	}

	claims := &bearerClaims{}
	claims.Subject = idToken.Subject
	claims.Roles = make(map[string][]string, len(accClaims.ResourceAccess) + 1)
	claims.Roles[config.GlobalRoleKey] = accClaims.RealmAccess.Roles
	for k, v := range accClaims.ResourceAccess {
		claims.Roles[k] = v.Roles
	}

	return claims, nil
}
