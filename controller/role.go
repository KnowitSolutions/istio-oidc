package controller

// Leader election
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;get;update

// Controllers
// +kubebuilder:rbac:groups=krsdev.app,resources=openidproviders,verbs=get
// +kubebuilder:rbac:groups=krsdev.app,resources=accesspolicies,verbs=get;list;update;watch
// +kubebuilder:rbac:groups=krsdev.app,resources=accesspolicies/status,verbs=update
// +kubebuilder:rbac:groups=networking.istio.io,resources=gateways,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters,verbs=create;get;list;update;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;get;list;watch

// Events
// +kubebuilder:rbac:groups="",resources=events,verbs=create
