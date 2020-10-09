package openidprovider

import (
	"context"
	"golang.org/x/oauth2"
)

type OpenIdProvider struct {
	cfg  openIdConfiguration
	maps []roleMapping
}

type openIdConfiguration struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JWKsURI               string `json:"jwks_uri"`
}

type from int

const (
	AccessToken = iota
	IdToken
)

var fromStrToConst = map[string]from{
	"":            AccessToken,
	"accesstoken": AccessToken,
	"idtoken":     IdToken,
}

type roleMapping struct {
	from   from
	prefix string
	path   []string
}

func (op OpenIdProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  op.cfg.AuthorizationEndpoint,
		TokenURL: op.cfg.TokenEndpoint,
	}
}

func (op OpenIdProvider) TokenData(ctx context.Context, tok oauth2.Token) (TokenData, error) {
	return extractTokenData(ctx, op, tok)
}

type TokenData struct {
	Subject string
	Roles   map[string][]string
}
