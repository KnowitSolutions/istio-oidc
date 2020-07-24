package accesspolicy

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"github.com/KnowitSolutions/istio-oidc/state"
	"github.com/KnowitSolutions/istio-oidc/state/accesspolicy"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type workerReconciler struct {
	client.Client
	state.AccessPolicyStore
}

func (r *workerReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	ctx = log.WithValues(ctx, "AccessPolicy", req.Namespace+"/"+req.Name, "leader", "false")

	ap := api.AccessPolicy{}
	err := r.Get(ctx, req.NamespacedName, &ap)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting AccessPolicy")
	}

	err = r.reconcileAuth(ctx, &ap)
	return reconcile.Result{}, err
}

func (r *workerReconciler) reconcileAuth(ctx context.Context, ap *api.AccessPolicy) error {
	if ap.DeletionTimestamp.IsZero() {
		credName := ap.Spec.OIDC.CredentialsSecret.Name
		credKey := types.NamespacedName{Namespace: ap.Namespace, Name: credName}
		cred := core.Secret{}
		err := r.Get(ctx, credKey, &cred)
		if err != nil {
			return errors.Wrap(err, "failed getting credentials Secret")
		}

		cfg, err := accesspolicy.NewAccessPolicy(ap, &cred)
		if err != nil {
			log.Error(ctx, err, "Invalid AccessPolicy")
			return nil
		}

		log.Info(ctx, nil, "Updating OIDC settings")
		r.UpdateAccessPolicy(ctx, cfg)
	} else {
		log.Info(ctx, nil, "Deleting OIDC settings")
		r.DeleteAccessPolicy(ctx, ap.Name)
	}

	return nil
}
