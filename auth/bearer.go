package auth

import (
	"context"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/logging/errors"
)

type bearerClaims struct {
	expirable
	jwt.Claims
	Roles []string `json:"rol"`
}

func makeBearerClaims(ctx context.Context, req *request, tok *oauth2.Token) (*bearerClaims, error) {
	accTok, idTok, err := req.accessPolicyHelper.ExtractTokens(ctx, tok)
	if err != nil {
		return nil, errors.Wrap(err, "failed making bearer claims")
	}

	count := len(accTok.RealmAccess.Roles)
	for _, v := range accTok.ResourceAccess {
		count += len(v.Roles)
	}

	claims := &bearerClaims{}
	claims.Subject = idTok.Subject
	claims.Roles = make([]string, 0, count)
	for _, v := range accTok.RealmAccess.Roles {
		claims.Roles = append(claims.Roles, v)
	}
	for k, v := range accTok.ResourceAccess {
		for _, v := range v.Roles {
			claims.Roles = append(claims.Roles, k+"/"+v)
		}
	}

	return claims, nil
}
