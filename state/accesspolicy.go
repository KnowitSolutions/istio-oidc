package state

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"istio-keycloak/api"
	"istio-keycloak/logging/errors"
	core "k8s.io/api/core/v1"
	"net/url"
	"strings"
)

const (
	AccessPolicyKey = "policy"
	RolesKey        = "roles"
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
	ClientId     string
	ClientSecret string
	Callback     url.URL
}

type Routes map[string]Route

type Route struct {
	EnableAuthz bool
	Roles       Roles
}

type Roles []string

type accessPolicySpecStatus struct {
	spec   api.AccessPolicySpec
	status api.AccessPolicyStatus
}
type accessPolicyOIDC api.AccessPolicyOIDC
type accessPolicyRoute api.AccessPolicyRoute

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
