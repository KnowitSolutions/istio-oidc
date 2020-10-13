package api

import (
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"regexp"
)

// +kubebuilder:object:root=true
type AccessPolicyList struct {
	meta.TypeMeta `json:",inline"`
	meta.ListMeta `json:"metadata"`

	Items []AccessPolicy `json:"items"`
}

// +kubebuilder:resource:path=accesspolicies,shortName=ap
// +kubebuilder:printcolumn:name=Gateway,type=string,JSONPath=.spec.gateway
// +kubebuilder:printcolumn:name=OpenID Provider,type=string,JSONPath=.spec.oidc.provider
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AccessPolicy struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata"`

	Spec AccessPolicySpec `json:"spec"`
	// +kubebuilder:validation:Optional
	Status AccessPolicyStatus `json:"status,omitempty"`
}

func (in *AccessPolicy) Validate() []error {
	return in.Spec.Validate([]error{})
}

func (in *AccessPolicy) Normalize() {
	in.Spec.Normalize()
}

// +kubebuilder:object:generate=true
type AccessPolicySpec struct {
	Gateway string           `json:"gateway"`
	OIDC    AccessPolicyOIDC `json:"oidc"`

	// +kubebuilder:validation:Optional
	Routes []AccessPolicyRoute `json:"routes,omitempty"`
}

func (in *AccessPolicySpec) Validate(errs []error) []error {
	errs = in.OIDC.Validate(errs)

	names := make(map[string]struct{}, len(in.Routes))
	for _, r := range in.Routes {
		_, ok := names[r.Name]
		if ok {
			err := errors.New("duplicate route name", "name", r.Name)
			errs = append(errs, err)
		} else {
			names[r.Name] = struct{}{}
		}
	}

	return errs
}

func (in *AccessPolicySpec) Normalize() {
	in.OIDC.Normalize()
}

type AccessPolicyOIDC struct {
	// +kubebuilder:validation:Pattern=`^([a-z-]+/)?[a-z-.]+$`
	Provider          string                            `json:"provider"`
	CredentialsSecret AccessPolicyOIDCCredentialsSecret `json:"credentialsSecretRef"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^\/[A-Za-z0-9\-._~!$&'()*+,;=:@\/%]*$|^$`
	CallbackPath string `json:"callbackPath"`
}

func (in *AccessPolicyOIDC) Validate(errs []error) []error {
	re := regexp.MustCompile("^([a-z-]+/)?[a-z-.]+$")
	ok := re.MatchString(in.Provider)
	if !ok {
		err := errors.New("invalid OpenID provider")
		errs = append(errs, err)
	}

	_, err := url.Parse(in.CallbackPath)
	if err != nil {
		err = errors.Wrap(err, "invalid callback path")
		errs = append(errs, err)
	}

	return errs
}

func (in *AccessPolicyOIDC) Normalize() {
	in.CredentialsSecret.Normalize()

	if in.CallbackPath == "" {
		in.CallbackPath = "/odic/callback"
	}
}

type AccessPolicyOIDCCredentialsSecret struct {
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	ClientIDKey string `json:"clientIDKey"`
	// +kubebuilder:validation:Optional
	ClientSecretKey string `json:"clientSecretKey"`
	// +kubebuilder:validation:Optional
	TokenSecretKey string `json:"tokenSecretKey"`
}

func (in *AccessPolicyOIDCCredentialsSecret) Normalize() {
	if in.ClientIDKey == "" {
		in.ClientIDKey = "clientID"
	}

	if in.ClientSecretKey == "" {
		in.ClientSecretKey = "clientSecret"
	}

	if in.TokenSecretKey == "" {
		in.TokenSecretKey = "tokenKey"
	}
}

// +kubebuilder:object:generate=true
type AccessPolicyRoute struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	Roles []string `json:"roles,omitempty"`
	// +kubebuilder:validation:Optional
	Headers []AccessPolicyRouteHeader `json:"headers,omitempty"`
	// +kubebuilder:validation:Optional
	DisableEnforcement bool `json:"disableEnforcement,omitempty"`
}

// +kubebuilder:object:generate=true
type AccessPolicyRouteHeader struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
	// +kubebuilder:validation:Optional
	Value string `json:"value,omitempty"`
}

// +kubebuilder:object:generate=true
type AccessPolicyStatus struct {
	// +kubebuilder:validation:Optional
	Ingress AccessPolicyStatusIngress `json:"ingress,omitempty"`
	// +kubebuilder:validation:Optional
	VirtualHosts []string `json:"virtualHosts,omitempty"`
}

func (in *AccessPolicyStatus) GetIngress() *AccessPolicyStatusIngress {
	if in == nil {
		return nil
	} else {
		return &in.Ingress
	}
}

// +kubebuilder:object:generate=true
type AccessPolicyStatusIngress struct {
	Selector map[string]string `json:"selector"`
}

func (in *AccessPolicyStatusIngress) GetSelector() map[string]string {
	if in == nil {
		return nil
	} else {
		return in.Selector
	}
}
