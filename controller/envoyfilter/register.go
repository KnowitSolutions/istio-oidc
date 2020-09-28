package envoyfilter

import (
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/controller/predicate"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func Register(mgr ctrl.Manager) error {
	scheme := mgr.GetScheme()
	err := istionetworking.AddToScheme(scheme)
	if err != nil {
		return errors.Wrap(err, "failed making EnvoyFilter controller")
	}

	err = register(mgr)
	if err != nil {
		return errors.Wrap(err, "failed making EnvoyFilter controller")
	}

	return nil
}

func register(mgr ctrl.Manager) error {
	r := reconciler{mgr.GetClient()}
	opts := controller.Options{Reconciler: &r}
	c, err := controller.NewUnmanaged("envoyfilter", mgr, opts)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &istionetworking.EnvoyFilter{}},
		&handler.EnqueueRequestForObject{},
		&predicate.GenerationChangedPredicate{},
		&predicate.InNamespace{Namespace: config.Controller.IstioRootNamespace},
		&predicate.HasLabels{Labels: config.Controller.EnvoyFilterLabels})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &api.AccessPolicy{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: &mapper{mgr.GetClient()}},
		&predicate.ResourceVersionChangedPredicate{})
	if err != nil {
		return err
	}

	return mgr.Add(c)
}
