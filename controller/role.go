package controller

//go:generate controller-gen rbac:roleName=istio-oidc output:dir=.
// +kubebuilder:rbac:groups=krsdev.app,resources=accesspolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=krsdev.app,resources=accesspolicies/status,verbs=update
// +kubebuilder:rbac:groups=networking.istio.io/v1alpha3,resources=gateways,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.istio.io/v1alpha3,resources=envoyfilters,verbs=create;get;list;update;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;get;list;watch
