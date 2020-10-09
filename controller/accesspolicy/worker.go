package accesspolicy

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"github.com/KnowitSolutions/istio-oidc/state/accesspolicy"
	"github.com/KnowitSolutions/istio-oidc/state/openidprovider"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type workerReconciler struct {
	client.Client
	record.EventRecorder
	AccessPolicies accesspolicy.Store
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

	if ap.DeletionTimestamp.IsZero() {
		err = r.reconcileAuth(ctx, &ap)
	} else {
		err = r.deleteAuth(ctx, &ap)
	}

	return reconcile.Result{}, err
}

func (r *workerReconciler) reconcileAuth(ctx context.Context, ap *api.AccessPolicy) error {
	re := regexp.MustCompile(`^(?:([a-z-]+)/)?([a-z-.]+)$`)
	opKeyParts := re.FindStringSubmatch(ap.Spec.OIDC.Provider)
	if opKeyParts == nil {
		log.Error(ctx, nil, "OpenID provider is not a valid identifier")
		r.Event(ap, "Warning", "InvalidProvider", "OpenID provider is not a valid identifier")
		return nil
	} else if opKeyParts[1] == "" {
		opKeyParts[1] = ap.Namespace
	}

	opKey := types.NamespacedName{Namespace: opKeyParts[1], Name: opKeyParts[2]}
	op := api.OpenIDProvider{}
	err := r.Get(ctx, opKey, &op)
	if err != nil {
		log.Error(ctx, err, "Failed getting OpenID provider")
		r.Event(ap, "Warning", "MissingProvider", "Failed getting OpenID provider")
		return nil
	}

	credName := ap.Spec.OIDC.CredentialsSecret.Name
	credKey := types.NamespacedName{Namespace: ap.Namespace, Name: credName}
	cred := core.Secret{}
	err = r.Get(ctx, credKey, &cred)
	if err != nil {
		log.Error(ctx, err, "Failed getting credentials secret")
		r.Event(ap, "Warning", "MissingCredentials", "Failed getting credentials secret")
		return nil
	}

	newAp, err := accesspolicy.New(ap, &cred)
	if err != nil {
		log.Error(ctx, err, "Invalid AccessPolicy")
		r.Event(ap, "Warning", "Invalid", "Invalid AccessPolicy")
		return nil
	}

	newOp, err := openidprovider.New(ctx, &op)
	if err != nil {
		log.Error(ctx, err, "Invalid OpenIDProvider")
		r.Event(&op, "Warning", "Invalid", "Invalid OpenIDProvider")
		return nil
	}
	newAp.Oidc.Provider = newOp

	log.Info(ctx, nil, "Storing OIDC settings")
	r.AccessPolicies.Update(ctx, newAp)

	return nil
}

func (r *workerReconciler) deleteAuth(ctx context.Context, ap *api.AccessPolicy) error {
	log.Info(ctx, nil, "Deleting OIDC settings")
	r.AccessPolicies.Delete(ctx, ap.Name)

	return nil
}