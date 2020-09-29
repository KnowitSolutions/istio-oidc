package secret

import (
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/controller/predicate"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"github.com/KnowitSolutions/istio-oidc/state"
	core "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func Register(mgr ctrl.Manager, keyStore state.KeyStore) error {
	scheme := mgr.GetScheme()
	err := core.AddToScheme(scheme)
	if err != nil {
		return errors.Wrap(err, "failed making Secret controller")
	}

	err = registerLeader(mgr)
	if err != nil {
		return errors.Wrap(err, "failed making Secret controller")
	}

	err = registerWorker(mgr, keyStore)
	if err != nil {
		return errors.Wrap(err, "failed making Secret controller")
	}

	return nil
}

func registerLeader(mgr ctrl.Manager) error {
	r := leaderReconciler{mgr.GetClient()}
	opts := controller.Options{Reconciler: &r}
	c, err := controller.NewUnmanaged("secret-leader", mgr, opts)
	if err != nil {
		return err
	}

	secret := core.Secret{}
	secret.Namespace = config.Controller.TokenKeyNamespace
	secret.Name = config.Controller.TokenKeyName
	src := make(chan event.GenericEvent, 1)
	src <- event.GenericEvent{Meta: &secret, Object: &secret}
	err = c.Watch(
		&source.Channel{Source: src},
		&handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &core.Secret{}},
		&handler.EnqueueRequestForObject{},
		&predicate.ResourceVersionChangedPredicate{},
		&predicate.InNamespace{Namespace: config.Controller.TokenKeyNamespace},
		&predicate.IsNamed{Name: config.Controller.TokenKeyName})
	if err != nil {
		return err
	}

	return mgr.Add(c)
}

func registerWorker(mgr ctrl.Manager, keyStore state.KeyStore) error {
	r := workerReconciler{mgr.GetClient(), keyStore}
	opts := controller.Options{Reconciler: &r}
	c, err := controller.NewUnmanaged("secret-worker", mgr, opts)
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &core.Secret{}},
		&handler.EnqueueRequestForObject{},
		&predicate.ResourceVersionChangedPredicate{},
		&predicate.InNamespace{Namespace: config.Controller.TokenKeyNamespace},
		&predicate.IsNamed{Name: config.Controller.TokenKeyName})
	if err != nil {
		return err
	}

	return mgr.Add(workerController{c})
}

type workerController struct {
	controller.Controller
}

func (workerController) NeedLeaderElection() bool {
	return false
}
