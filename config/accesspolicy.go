package config

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"istio-keycloak/api/v1"
	"istio-keycloak/logging/errors"
	core "k8s.io/api/core/v1"
	"strings"
)

const (
	AccessPolicyKey = "policy"
	RolesKey        = "roles"
	GlobalRouteKey  = "*"
	GlobalRoleKey   = ""
)

type AccessPolicy struct {
	Name   string
	Realm  string
	OIDC   OIDC
	Global Route
	Routes Routes
}

type OIDC struct {
	ClientID     string
	ClientSecret string
	CallbackPath string
}

type Routes map[string]Route

type Route struct {
	EnableAuthz bool
	Roles       Roles
}

type Roles map[string][]string

type accessPolicySpec api.AccessPolicySpec
type accessPolicyOIDC api.AccessPolicyOIDC
type accessPolicyRoute api.AccessPolicyRoute

func NewAccessPolicy(ap *api.AccessPolicy, secret *core.Secret) *AccessPolicy {
	spec := accessPolicySpec(ap.Spec)
	return spec.convert(ap.Name, secret)
}

func (ap *accessPolicySpec) normalize() {
	if ap.GlobalRolesKey == "" {
		ap.GlobalRolesKey = "*"
	}
}

func (ap *accessPolicySpec) convert(name string, secret *core.Secret) *AccessPolicy {
	ap.normalize()

	oidc := accessPolicyOIDC(ap.OIDC)
	global := accessPolicyRoute(ap.Routes[GlobalRouteKey])

	routes := make(Routes, len(ap.Routes)-1)
	for k, route := range ap.Routes {
		if k != GlobalRouteKey {
			route := accessPolicyRoute(route)
			routes[k] = route.convert(ap.GlobalRolesKey)
		}
	}

	return &AccessPolicy{
		Name:   name,
		Realm:  ap.Realm,
		OIDC:   oidc.convert(secret),
		Global: global.convert(ap.GlobalRolesKey),
		Routes: routes,
	}
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

func (apo *accessPolicyOIDC) convert(secret *core.Secret) OIDC {
	apo.normalize()
	return OIDC{
		ClientID:     string(secret.Data[apo.CredentialsSecret.ClientIDKey]),
		ClientSecret: string(secret.Data[apo.CredentialsSecret.ClientSecretKey]),
		CallbackPath: apo.CallbackPath,
	}
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
