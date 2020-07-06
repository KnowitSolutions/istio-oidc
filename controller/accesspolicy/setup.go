package accesspolicy

import (
	"istio-keycloak/api"
	"istio-keycloak/config"
	"istio-keycloak/controller/predicate"
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func New(mgr ctrl.Manager, apStore state.AccessPolicyStore) error {
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

	c := controller{mgr.GetClient(), mgr.GetScheme(), apStore}
	err = builder.
		ControllerManagedBy(mgr).
		For(
			&api.AccessPolicy{},
			builder.WithPredicates(&predicate.ResourceVersionChangedPredicate{})).
		Watches(
			&source.Kind{Type: &core.Secret{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: newSecretMapper(mgr)},
			builder.WithPredicates(&predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &istionetworking.Gateway{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: newGatewayMapper(mgr)},
			builder.WithPredicates(&predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &istionetworking.EnvoyFilter{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: newEnvoyFilterMapper(mgr)},
			builder.WithPredicates(
				&predicate.GenerationChangedPredicate{},
				&predicate.InNamespace{Namespace: config.Controller.IstioRootNamespace},
				&predicate.HasLabels{Labels: config.Controller.EnvoyFilterLabels})).
		Complete(&c)
	if err != nil {
		return errors.Wrap(err, "failed making AccessPolicy controller")
	}

	return nil
}
