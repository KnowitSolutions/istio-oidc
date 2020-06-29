package state

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"istio-keycloak/api/v1"
	"istio-keycloak/logging/errors"
	core "k8s.io/api/core/v1"
	"net/url"
	"strings"
)

const (
	AccessPolicyKey = "policy"
	RolesKey        = "roles"
	GlobalRouteKey  = "*"
	GlobalRoleKey   = ""
)

type AccessPolicy struct {
	Name         string
	Realm        string
	OIDC         OIDC
	VirtualHosts []string
	Global       Route
	Routes       Routes
}

type OIDC struct {
	ClientId     string
	ClientSecret string
	Callback     url.URL
}

type Routes map[string]Route

type Route struct {
	EnableAuthz bool
	Roles       Roles
}

type Roles map[string][]string

type accessPolicySpecStatus struct {
	spec   api.AccessPolicySpec
	status api.AccessPolicyStatus
}
type accessPolicyOIDC api.AccessPolicyOIDC
type accessPolicyRoute api.AccessPolicyRoute

func NewAccessPolicy(ap *api.AccessPolicy, secret *core.Secret) (*AccessPolicy, error) {
	spec := accessPolicySpecStatus{ap.Spec, ap.Status}
	return spec.convert(ap.Name, secret)
}

func (ap *accessPolicySpecStatus) normalize() {
	if ap.spec.GlobalRolesKey == "" {
		ap.spec.GlobalRolesKey = "*"
	}
}

func (ap *accessPolicySpecStatus) convert(name string, secret *core.Secret) (*AccessPolicy, error) {
	ap.normalize()

	oidc := accessPolicyOIDC(ap.spec.OIDC)
	global := accessPolicyRoute(ap.spec.Routes[GlobalRouteKey])

	routes := make(Routes, len(ap.spec.Routes)-1)
	for k, route := range ap.spec.Routes {
		if k != GlobalRouteKey {
			route := accessPolicyRoute(route)
			routes[k] = route.convert(ap.spec.GlobalRolesKey)
		}
	}

	oidcCfg, err := oidc.convert(secret)
	if err != nil {
		return nil, err
	}

	return &AccessPolicy{
		Name:         name,
		Realm:        ap.spec.Realm,
		OIDC:         oidcCfg,
		VirtualHosts: ap.status.VirtualHosts,
		Global:       global.convert(ap.spec.GlobalRolesKey),
		Routes:       routes,
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

func (apo *accessPolicyOIDC) convert(secret *core.Secret) (OIDC, error) {
	apo.normalize()

	cb, err := url.Parse(apo.CallbackPath)
	if err != nil {
		return OIDC{}, nil
	}

	var clientId, clientSecret string
	if secret != nil {
		clientId = string(secret.Data[apo.CredentialsSecret.ClientIDKey])
		clientSecret = string(secret.Data[apo.CredentialsSecret.ClientSecretKey])
	}

	return OIDC{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		Callback:     *cb,
	}, nil
}

func (apr *accessPolicyRoute) normalize(globalRolesKey string) {
	if apr.Roles == nil {
		return
	}

	global, ok := apr.Roles[globalRolesKey]
	if !ok {
		return
	}

	delete(apr.Roles, globalRolesKey)
	apr.Roles[GlobalRoleKey] = global
}

func (apr *accessPolicyRoute) convert(globalRolesKey string) Route {
	apr.normalize(globalRolesKey)
	return Route{
		EnableAuthz: !apr.DisableAccessPolicy,
		Roles:       apr.Roles,
	}
}

func (r *Roles) Encode() (string, error) {
	buf := bytes.NewBuffer(nil)
	b64 := base64.NewEncoder(base64.StdEncoding, buf)
	enc := gob.NewEncoder(b64)
	err := enc.Encode(r)
	_ = b64.Close()
	return buf.String(), err
}

func (r *Roles) Decode(str string) error {
	buf := strings.NewReader(str)
	b64 := base64.NewDecoder(base64.StdEncoding, buf)
	dec := gob.NewDecoder(b64)
	return errors.Wrap(dec.Decode(r), "unable to decode roles")
}
