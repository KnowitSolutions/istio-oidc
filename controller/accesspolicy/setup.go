package accesspolicy

import (
	"istio-keycloak/api"
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func New(mgr ctrl.Manager, oidcComms state.OidcCommunicatorStore) error {
	scheme := mgr.GetScheme()
	err := core.AddToScheme(scheme)
	if err != nil {
		return errors.Wrap(err, "failed making AccessPolicy controller")
	}
	err = istionetworking.AddToScheme(scheme)
	if err != nil {
		return errors.Wrap(err, "failed making AccessPolicy controller")
	}
	err = api.AddToScheme(scheme)
	if err != nil {
		return errors.Wrap(err, "failed making AccessPolicy controller")
	}

	c := controller{mgr.GetClient(), mgr.GetScheme(), oidcComms}
	err = builder.
		ControllerManagedBy(mgr).
		For(
			&api.AccessPolicy{},
			builder.WithPredicates(&predicate.ResourceVersionChangedPredicate{})).
		Watches(
			&source.Kind{Type: &core.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: &mapper{mgr.GetClient()}},
			builder.WithPredicates(&predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &istionetworking.Gateway{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: &mapper{mgr.GetClient()}},
			builder.WithPredicates(&predicate.GenerationChangedPredicate{})).
		Owns(
			&istionetworking.EnvoyFilter{},
			builder.WithPredicates(&predicate.GenerationChangedPredicate{})).
		Complete(&c)
	if err != nil {
		return errors.Wrap(err, "failed making AccessPolicy controller")
	}

	return nil
}
