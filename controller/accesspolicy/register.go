package accesspolicy

import (
	"istio-keycloak/api"
	"istio-keycloak/config"
	"istio-keycloak/controller/predicate"
	"istio-keycloak/log/errors"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func Register(mgr ctrl.Manager, apStore state.AccessPolicyStore) error {
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

	err = registerLeader(mgr)
	if err != nil {
		return errors.Wrap(err, "failed making AccessPolicy controller")
	}

	err = registerWorker(mgr, apStore)
	if err != nil {
		return errors.Wrap(err, "failed making AccessPolicy controller")
	}

	return nil
}

func registerLeader(mgr ctrl.Manager) error {
	r := leaderReconciler{mgr.GetClient()}
	opts := controller.Options{Reconciler: &r}
	c, err := controller.NewUnmanaged("accesspolicy-leader", mgr, opts)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &api.AccessPolicy{}},
		&handler.EnqueueRequestForObject{},
		&predicate.ResourceVersionChangedPredicate{})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &istionetworking.Gateway{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: newGatewayMapper(mgr)},
		&predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &istionetworking.EnvoyFilter{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: newEnvoyFilterMapper(mgr)},
		&predicate.GenerationChangedPredicate{},
		&predicate.InNamespace{Namespace: config.Controller.IstioRootNamespace},
		&predicate.HasLabels{Labels: config.Controller.EnvoyFilterLabels})
	if err != nil {
		return err
	}

	return mgr.Add(c)
}

func registerWorker(mgr ctrl.Manager, apStore state.AccessPolicyStore) error {
	r := workerReconciler{mgr.GetClient(), apStore}
	opts := controller.Options{Reconciler: &r}
	c, err := controller.NewUnmanaged("accesspolicy-worker", mgr, opts)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &api.AccessPolicy{}},
		&handler.EnqueueRequestForObject{},
		&predicate.ResourceVersionChangedPredicate{})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &core.Secret{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: newSecretMapper(mgr)},
		&predicate.GenerationChangedPredicate{})
	if err != nil {
		return err
	}

	return mgr.Add(workerController{c})
}

type workerController struct {
	controller.Controller
}

func (_ workerController) NeedLeaderElection() bool {
	return false
}
