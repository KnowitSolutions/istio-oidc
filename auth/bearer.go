package auth

import (
	"context"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
)

type bearerClaims struct {
	expirableImpl
	jwt.Claims
	Roles map[string][]string `json:"rol"`
}

func makeBearerClaims(ctx context.Context, req *request, tok *oauth2.Token) (*bearerClaims, error) {
	accTok, idTok, err := req.oidc.ExtractTokens(ctx, tok)
	if err != nil {
		return nil, errors.Wrap(err, "failed making bearer claims")
	}

	claims := &bearerClaims{}
	claims.Subject = idTok.Subject
	claims.Roles = make(map[string][]string, len(accTok.ResourceAccess)+1)
	claims.Roles[state.GlobalRoleKey] = accTok.RealmAccess.Roles
	for k, v := range accTok.ResourceAccess {
		claims.Roles[k] = v.Roles
	}

	return claims, nil
}
