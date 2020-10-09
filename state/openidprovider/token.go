package openidprovider

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"time"
)

func extractTokenData(ctx context.Context, op OpenIdProvider, tok oauth2.Token) (TokenData, error) {
	jwks := jose.JSONWebKeySet{}
	err := doJsonRequest(ctx, op.cfg.JWKsURI, &jwks)
	if err != nil {
		return TokenData{}, errors.Wrap(err, "unable to retrieve JWKs")
	}

	at := make(map[string]interface{}, 0)
	err = claims(tok.AccessToken, jwks, &at)
	if err != nil {
		return TokenData{}, errors.Wrap(err, "unable to get access token claims")
	}

	idt := make(map[string]interface{}, 0)
	err = claims(tok.Extra("id_token").(string), jwks, &idt)
	if err != nil {
		return TokenData{}, errors.Wrap(err, "unable to get ID token claims")
	}

	roles := make(map[string][]string, 0)
	for _, rm := range op.maps {
		var tok map[string]interface{}
		switch rm.from {
		case AccessToken:
			tok = at
		case IdToken:
			tok = idt
		}

		extracted, err := extractRoles(rm.path, tok)
		if err != nil {
			err = errors.Wrap(err, "failed extracting roles")
			return TokenData{}, err
		}

		roles[rm.prefix] = append(roles[rm.prefix], extracted...)
	}

	sub, _ := idt["sub"].(string)
	return TokenData{
		Subject: sub,
		Roles:   roles,
	}, nil
}

func claims(tok string, jwks jose.JSONWebKeySet, claims interface{}) error {
	parsed, err := jwt.ParseSigned(tok)
	if err != nil {
		return errors.Wrap(err, "failed parsing token", "token", tok)
	}

	def := &jwt.Claims{}
	err = parsed.Claims(&jwks, def)
	if err != nil {
		return errors.Wrap(err, "failed deserializing default claims", "token", tok)
	}

	exp := jwt.Expected{Time: time.Now()}
	err = def.ValidateWithLeeway(exp, 0)
	if err != nil {
		return errors.Wrap(err, "failed validating token", "token", tok)
	}

	err = parsed.Claims(&jwks, claims)
	if err != nil {
		return errors.Wrap(err, "failed deserializing custom claims", "token", tok)
	}

	return nil
}
