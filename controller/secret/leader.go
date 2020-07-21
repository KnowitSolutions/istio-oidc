package secret

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const dataKey = "secret"

type leaderReconciler struct {
	client.Client
}

func (r *leaderReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	ctx = log.WithValues(ctx, "Secret", req.Namespace+"/"+req.Name, "leader", "true")

	secret := core.Secret{}
	err := r.Get(ctx, req.NamespacedName, &secret)
	if err == nil {
		log.Info(ctx, nil, "Secret already exits")
		return reconcile.Result{}, nil
	} else if !apierrors.IsNotFound(err) {
		return reconcile.Result{}, errors.Wrap(err, "failed getting Secret")
	}


	key := make([]byte, sha512.Size)
	_, err = rand.Read(key)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed creating key")
	}

	secret.Namespace = req.Namespace
	secret.Name = req.Name
	secret.Data = map[string][]byte{dataKey: key}
	err = r.Create(ctx, &secret)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed creating Secret")
	}

	log.Info(ctx, nil, "Created secret")
	return reconcile.Result{}, nil
}
