package api

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
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
// +kubebuilder:subresource:status
type AccessPolicy struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata"`

	Spec AccessPolicySpec `json:"spec"`
	// +kubebuilder:validation:Optional
	Status AccessPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type AccessPolicySpec struct {
	Gateway string `json:"gateway"`

	Realm string           `json:"realm"`
	OIDC  AccessPolicyOIDC `json:"oidc"`

	// +kubebuilder:validation:Optional
	Routes []AccessPolicyRoute `json:"routes,omitempty"`
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
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	Roles [][]string `json:"roles,omitempty"`
	// +kubebuilder:validation:Optional
	Headers []AccessPolicyRouteHeader `json:"headers,omitempty"`
	// +kubebuilder:validation:Optional
	DisableAccessPolicy bool `json:"disableAccessPolicy,omitempty"`
}

// +kubebuilder:object:generate=true
type AccessPolicyRouteHeader struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Value string     `json:"value"`
	Roles [][]string `json:"roles"`
}

// +kubebuilder:object:generate=true
type AccessPolicyStatus struct {
	// +kubebuilder:validation:Optional
	Ingress AccessPolicyStatusIngress `json:"ingress,omitempty"`
	// +kubebuilder:validation:Optional
	VirtualHosts []string `json:"virtualHosts,omitempty"`
}

// +kubebuilder:object:generate=true
type AccessPolicyStatusIngress struct {
	Selector map[string]string `json:"selector"`
}
