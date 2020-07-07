package accesspolicy

import (
	"fmt"
	"istio-keycloak/api"
	"istio-keycloak/log/errors"
	core "k8s.io/api/core/v1"
	"net/url"
)

type accessPolicySpecStatus struct {
	spec   api.AccessPolicySpec
	status api.AccessPolicyStatus
}
type accessPolicyOIDC api.AccessPolicyOIDC
type accessPolicyRoute api.AccessPolicyRoute
type accessPolicyRoles []string
type accessPolicyRouteHeaders []api.AccessPolicyRouteHeader
type accessPolicyRouteHeader api.AccessPolicyRouteHeader

func NewAccessPolicy(ap *api.AccessPolicy, secret *core.Secret) (*AccessPolicy, error) {
	spec := accessPolicySpecStatus{ap.Spec, ap.Status}
	name := fmt.Sprintf("%s/%s", ap.Namespace, ap.Name)
	return spec.convert(name, secret)
}

func (ap *accessPolicySpecStatus) convert(name string, secret *core.Secret) (*AccessPolicy, error) {
	oidc := accessPolicyOIDC(ap.spec.OIDC)

	var defRoute Route
	routes := make(Routes, len(ap.spec.Routes)-1)
	for _, route := range ap.spec.Routes {
		route := accessPolicyRoute(route)
		if route.Name == "" {
			defRoute = route.convert()
		} else {
			routes[route.Name] = route.convert()
		}
	}

	oidcCfg, err := oidc.convert(secret)
	if err != nil {
		return nil, err
	}

	return &AccessPolicy{
		Name:         name,
		Realm:        ap.spec.Realm,
		Oidc:         oidcCfg,
		Default:      defRoute,
		Routes:       routes,
		VirtualHosts: ap.status.VirtualHosts,
	}, nil
}

func (apo *accessPolicyOIDC) normalize() {
	if apo.CredentialsSecret.ClientIDKey == "" {
		apo.CredentialsSecret.ClientIDKey = "clientID"
	}
	if apo.CredentialsSecret.ClientSecretKey == "" {
		apo.CredentialsSecret.ClientSecretKey = "clientSecret"
	}
	if apo.CallbackPath == "" {
		apo.CallbackPath = "/odic/callback"
	}
}

func (apo *accessPolicyOIDC) convert(secret *core.Secret) (Oidc, error) {
	apo.normalize()

	cb, err := url.Parse(apo.CallbackPath)
	if err != nil {
		return Oidc{}, err
	}

	var clientId, clientSecret string
	if secret != nil {
		clientIdBytes, ok1 := secret.Data[apo.CredentialsSecret.ClientIDKey]
		clientSecretBytes, ok2 := secret.Data[apo.CredentialsSecret.ClientSecretKey]

		if !ok1 || !ok2 {
			return Oidc{}, errors.New("failed extracting client ID and secret")
		} else {
			clientId = string(clientIdBytes)
			clientSecret = string(clientSecretBytes)
		}
	}

	return Oidc{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		Callback:     *cb,
	}, nil
}

func (apr *accessPolicyRoute) convert() Route {
	roles := accessPolicyRoles(apr.Roles)
	headers := accessPolicyRouteHeaders(apr.Headers)

	return Route{
		EnableAuthz: !apr.DisableAccessPolicy,
		Roles:       roles.convert(),
		Headers:     headers.convert(),
	}
}
func (apr *accessPolicyRoles) convert() Roles {
	roles := make(Roles, len(*apr))
	for i := range *apr {
		roles[i] = (*apr)[i]
	}
	return roles
}

func (aprh *accessPolicyRouteHeaders) convert() Headers {
	headers := make(Headers, len(*aprh))
	for i, header := range *aprh {
		header := accessPolicyRouteHeader(header)
		headers[i] = header.convert()
	}
	return headers
}

func (aprh *accessPolicyRouteHeader) convert() Header {
	roles := accessPolicyRoles(aprh.Roles)

	return Header{
		Name:  aprh.Name,
		Value: aprh.Value,
		Roles: roles.convert(),
	}
}
