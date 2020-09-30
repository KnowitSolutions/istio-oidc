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

type OpenIDProviderSpec struct {
	Issuer string `json:"issuer"`
}
