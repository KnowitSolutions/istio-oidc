package api

import (
	"istio-keycloak/config"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

// +kubebuilder:object:root=true
type AccessPolicyList struct {
	meta.TypeMeta `json:",inline"`
	meta.ListMeta `json:"metadata"`

	Items []AccessPolicy `json:"items"`
}

// +kubebuilder:resource:path=accesspolicies,shortName=ap
// +kubebuilder:printcolumn:name=Gateway,type=string,JSONPath=.spec.gateway
// +kubebuilder:printcolumn:name=Realm,type=string,JSONPath=.spec.realm
// +kubebuilder:object:root=true
type AccessPolicy struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata"`

	Spec AccessPolicySpec `json:"spec"`
}

// +kubebuilder:object:generate=true
type AccessPolicySpec struct {
	Gateway string `json:"gateway"`

	Realm string           `json:"realm"`
	OIDC  AccessPolicyOIDC `json:"oidc"`

	// +kubebuilder:validation:Optional
	GlobalRolesKey string `json:"globalRolesKey"`

	// +kubebuilder:validation:Optional
	Routes map[string]AccessPolicyRoute `json:"routes"`
}

type AccessPolicyOIDC struct {
	CredentialsSecret AccessPolicyOIDCCredentialsSecret `json:"credentialsSecretRef"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^\/[A-Za-z0-9\-._~!$&'()*+,;=:@\/%]*$`
	CallbackPath string `json:"callbackPath"`
}

type AccessPolicyOIDCCredentialsSecret struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	ClientIDKey string `json:"clientIDKey"`
	// +kubebuilder:validation:Optional
	ClientSecretKey string `json:"clientSecretKey"`
}

// +kubebuilder:object:generate=true
type AccessPolicyRoute struct {
	// +kubebuilder:validation:Optional
	Roles map[string][]string `json:"roles"`
	// +kubebuilder:validation:Optional
	DisableAccessPolicy bool `json:"disableAccessPolicy"`
}

func (ap *AccessPolicy) CredentialsKey() types.NamespacedName {
	return types.NamespacedName{Namespace: ap.Namespace, Name: ap.Spec.OIDC.CredentialsSecret.Name}
}

func (ap *AccessPolicy) GatewayKey() types.NamespacedName {
	parts := strings.SplitN(ap.Spec.Gateway, "/", 2)
	if len(parts) == 2 {
		return types.NamespacedName{Namespace: parts[0], Name: parts[1]}
	} else {
		return types.NamespacedName{Namespace: ap.Namespace, Name: parts[0]}
	}
}

func (ap *AccessPolicy) ToConfig(secret *core.Secret) *config.Service {
	return ap.Spec.ToConfig(ap.Name, secret)
}

func (ap *AccessPolicySpec) normalize() {
	if ap.GlobalRolesKey == "" {
		ap.GlobalRolesKey = "*"
	}
}

func (ap *AccessPolicySpec) ToConfig(name string, secret *core.Secret) *config.Service {
	ap.normalize()
	global := ap.Routes[config.GlobalRouteKey]

	routes := make(config.Routes, len(ap.Routes)-1)
	for k, route := range ap.Routes {
		if k != config.GlobalRouteKey {
			routes[k] = route.ToConfig(ap.GlobalRolesKey)
		}
	}

	return &config.Service{
		Name:   name,
		Realm:  ap.Realm,
		OIDC:   ap.OIDC.ToConfig(secret),
		Global: global.ToConfig(ap.GlobalRolesKey),
		Routes: routes,
	}
}

func (apo *AccessPolicyOIDC) normalize() {
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

func (apo *AccessPolicyOIDC) ToConfig(secret *core.Secret) config.OIDC {
	apo.normalize()
	return config.OIDC{
		ClientID:     string(secret.Data[apo.CredentialsSecret.ClientIDKey]),
		ClientSecret: string(secret.Data[apo.CredentialsSecret.ClientSecretKey]),
		CallbackPath: apo.CallbackPath,
	}
}

func (apr *AccessPolicyRoute) normalize(globalRolesKey string) {
	if apr.Roles == nil {
		return
	}

	global, ok := apr.Roles[globalRolesKey]
	if !ok {
		return
	}

	delete(apr.Roles, globalRolesKey)
	apr.Roles[config.GlobalRoleKey] = global
}

func (apr *AccessPolicyRoute) ToConfig(globalRolesKey string) config.Route {
	apr.normalize(globalRolesKey)
	return config.Route{
		EnableAuthz: !apr.DisableAccessPolicy,
		Roles:       apr.Roles,
	}
}
