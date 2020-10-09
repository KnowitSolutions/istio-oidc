package accesspolicy

import (
	"github.com/KnowitSolutions/istio-oidc/state/openidprovider"
	"golang.org/x/oauth2"
	"net/url"
)

const (
	NameKey  = "accessPolicy"
	RouteKey = "route"
)

type AccessPolicy struct {
	Name         string
	Oidc         Oidc
	Default      Route
	Routes       Routes
	VirtualHosts []string
}

type Oidc struct {
	Provider     openidprovider.OpenIdProvider
	ClientId     string
	ClientSecret string
	TokenSecret  []byte
	Callback     url.URL
}

type Routes map[string]Route
type Route struct {
	EnableAuthz bool
	Roles       []string
	Headers     Headers
}

type Headers []Header
type Header struct {
	Name  string
	Value string
	Roles []string
}

func (oidc Oidc) IsCallback(url url.URL) bool {
	return url.Path == oidc.Callback.Path
}

func (oidc Oidc) OAuth2(url url.URL) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     oidc.ClientId,
		ClientSecret: oidc.ClientSecret,
		Endpoint:     oidc.Provider.Endpoint(),
		RedirectURL:  url.ResolveReference(&oidc.Callback).String(),
		Scopes:       []string{"openid"},
	}
}
