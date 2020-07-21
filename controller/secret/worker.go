package secret

import (
	"context"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"istio-keycloak/state"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type workerReconciler struct {
	client.Client
	state.KeyStore
}

func (r *workerReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	ctx = log.WithValues(ctx, "Secret", req.Namespace+"/"+req.Name, "leader", "false")

	secret := core.Secret{}
	err := r.Get(ctx, req.NamespacedName, &secret)
	if apierrors.IsNotFound(err) {
		log.Info(ctx, nil, "Missing Secret")
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting Secret")
	}

	r.UpdateKey(secret.Data[dataKey])
	log.Info(ctx, nil, "Updated token key")

	return reconcile.Result{}, nil
}
