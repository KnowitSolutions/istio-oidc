package envoyfilter

import (
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func New(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()
	err := istionetworking.AddToScheme(scheme)
	if err != nil {
		return errors.Wrap(err, "failed making EnvoyFilter controller")
	}

	c := controller{mgr.GetClient()}
	err = builder.
		ControllerManagedBy(mgr).
		For(
			&istionetworking.EnvoyFilter{},
			builder.WithPredicates(
				&predicate.GenerationChangedPredicate{},
				&inNamespace{Namespace: config.Controller.IstioRootNamespace},
				&hasPrefix{Prefix: config.Controller.EnvoyFilterNamePrefix})).
		Complete(&c)
	if err != nil {
		return errors.Wrap(err, "failed making EnvoyFilter controller")
	}

	return nil
}
