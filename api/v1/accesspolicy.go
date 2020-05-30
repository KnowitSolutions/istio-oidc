package api

import meta "k8s.io/apimachinery/pkg/apis/meta/v1"

// TODO: Defaults:
// AccessPolicy.OIDC.CredentialsSecret.ClientIDKey = "clientID"
// AccessPolicy.OIDC.CredentialsSecret.ClientSecretKey = "clientSecret"
// AccessPolicy.OIDC.CallbackPath = "/odic/callback"
// AccessPolicy.GlobalRoleNamespaceKey = "*"

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
	GlobalRoleNamespaceKey string `json:"globalRoleNamespaceKey"`

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
