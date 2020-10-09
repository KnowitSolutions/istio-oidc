package api

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
type OpenIDProviderList struct {
	meta.TypeMeta `json:",inline"`
	meta.ListMeta `json:"metadata"`

	Items []OpenIDProvider `json:"items"`
}

// +kubebuilder:resource:path=openidproviders
// +kubebuilder:printcolumn:name=Issuer,type=string,JSONPath=.spec.issuer
// +kubebuilder:object:root=true
type OpenIDProvider struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata"`

	Spec OpenIDProviderSpec `json:"spec"`
}

// +kubebuilder:object:generate=true
type OpenIDProviderSpec struct {
	Issuer string `json:"issuer"`
	// +kubebuilder:validation:Optional
	RoleMappings []OpenIDProviderRoleMapping `json:"roleMappings"`
}

type OpenIDProviderRoleMapping struct {
	// +kubebuilder:validation:Optional
	From   string `json:"from"`
	// +kubebuilder:validation:Optional
	Prefix string `json:"prefix"`
	Path   string `json:"path"`
}
