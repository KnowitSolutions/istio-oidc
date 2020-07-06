package accesspolicy

import (
	"context"
	"golang.org/x/oauth2"
	"net/url"
)

const (
	NameKey  = "accessPolicy"
	RouteKey = "route"
)

type AccessPolicy struct {
	Name         string
	Realm        string
	Oidc         Oidc
	Default      Route
	Routes       Routes
	VirtualHosts []string
}

type Oidc struct {
	oidcProvider
	ClientId     string
	ClientSecret string
	Callback     url.URL
}

type Routes map[string]Route
type Route struct {
	EnableAuthz bool
	Roles       RoleSet
	Headers     Headers
}

type RoleSet []Roles
type Roles []string

type Headers []Header
type Header struct {
	Name  string
	Value string
	Roles RoleSet
}

func (ap *AccessPolicy) UpdateOidcProvider(ctx context.Context) error {
	return ap.Oidc.updateOidcProvider(ctx, ap.Realm)
}

func (oidc Oidc) IsCallback(url url.URL) bool {
	return url.Path == oidc.Callback.Path
}

func (oidc Oidc) OAuth2(url url.URL) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     oidc.ClientId,
		ClientSecret: oidc.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  oidc.AuthorizationEndpoint,
			TokenURL: oidc.TokenEndpoint,
		},
		RedirectURL: url.ResolveReference(&oidc.Callback).String(),
		Scopes:      []string{"openid"},
	}
}
