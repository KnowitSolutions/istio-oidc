package accesspolicy

import (
	"context"
	"istio-keycloak/api"
	"istio-keycloak/config"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"istio-keycloak/state"
	"istio-keycloak/state/accesspolicy"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sync"
)

const finalizer = "finalizer.istio-keycloak"

type controller struct {
	client.Client
	*runtime.Scheme
	state.AccessPolicyStore
}

func (c *controller) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	ctx = log.WithValues(ctx, "AccessPolicy", req.Namespace+"/"+req.Name)

	ap := api.AccessPolicy{}
	err := c.Get(ctx, req.NamespacedName, &ap)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting AccessPolicy")
	}

	if ap.DeletionTimestamp.IsZero() && !contains(ap.Finalizers, finalizer) {
		log.Info(ctx, nil, "Adding finalizer")
		ap.Finalizers = append(ap.Finalizers, finalizer)
		if err := c.Update(ctx, &ap); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed adding AccessPolicy finalizer")
		}
	}

	var wg sync.WaitGroup
	var resErr, efErr, authErr error
	go func() {
		resErr = c.reconcileStatus(ctx, &ap)
		efErr = c.reconcileEnvoyFilter(ctx, &ap)
		wg.Done()
	}()
	go func() {
		authErr = c.reconcileAuth(ctx, &ap)
		wg.Done()
	}()
	wg.Add(2)
	wg.Wait()
	err = errors.Merge(resErr, efErr, authErr)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !ap.DeletionTimestamp.IsZero() && contains(ap.Finalizers, finalizer) {
		log.Info(ctx, nil, "Removing finalizer")
		ap.Finalizers = remove(ap.Finalizers, finalizer)
		if err := c.Update(ctx, &ap); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed removing AccessPolicy finalizer")
		}
	}

	return reconcile.Result{}, nil
}

func (c *controller) reconcileStatus(ctx context.Context, ap *api.AccessPolicy) error {
	if ap.DeletionTimestamp.IsZero() {
		gwName := ap.Spec.Gateway
		gwKey := types.NamespacedName{Namespace: ap.Namespace, Name: gwName}
		gw := istionetworking.Gateway{}
		err := c.Get(ctx, gwKey, &gw)
		if err != nil {
			return errors.Wrap(err, "failed getting Gateway")
		}

		ap.Status.Ingress.Selector = selector(&gw)
		ap.Status.VirtualHosts = virtualHosts(&gw)

		log.Info(ctx, nil, "Updating status")
		err = c.Status().Update(ctx, ap)
		if err != nil {
			return errors.Wrap(err, "failed updating AccessPolicy status")
		}
	}

	return nil
}

func (c *controller) reconcileEnvoyFilter(ctx context.Context, ap *api.AccessPolicy) error {
	if len(ap.Status.GetIngress().GetSelector()) == 0 {
		log.Info(ctx, nil, "Missing workload selector")
		return nil
	}

	allEfs := istionetworking.EnvoyFilterList{}
	err := c.List(ctx, &allEfs,
		client.InNamespace(config.Controller.IstioRootNamespace),
		client.MatchingLabels(config.Controller.EnvoyFilterLabels))
	if err != nil {
		return errors.Wrap(err, "failed listing EnvoyFilters")
	}

	efs := make([]*istionetworking.EnvoyFilter, 0, len(allEfs.Items))
	for i := range allEfs.Items {
		if reflect.DeepEqual(allEfs.Items[i].Spec.GetWorkloadSelector().GetLabels(), ap.Status.Ingress.Selector) {
			efs = append(efs, &allEfs.Items[i])
		}
	}

	if len(efs) == 0 {
		log.Info(ctx, nil, "Creating EnvoyFilter")
		err = c.Create(ctx, newEnvoyFilter(ap))
		if err != nil {
			return errors.Wrap(err, "failed creating EnvoyFilter")
		}
	} else {
		vals := log.MakeValues("EnvoyFilter", efs[0].Name)
		log.Info(ctx, vals, "Found EnvoyFilter")
	}

	return nil
}

func (c *controller) reconcileAuth(ctx context.Context, ap *api.AccessPolicy) error {
	if ap.DeletionTimestamp.IsZero() {
		credName := ap.Spec.OIDC.CredentialsSecret.Name
		credKey := types.NamespacedName{Namespace: ap.Namespace, Name: credName}
		cred := core.Secret{}
		err := c.Get(ctx, credKey, &cred)
		if err != nil {
			return errors.Wrap(err, "failed getting credentials Secret")
		}

		cfg, err := accesspolicy.NewAccessPolicy(ap, &cred)
		if err != nil {
			log.Error(ctx, err, "Invalid AccessPolicy")
			return nil
		}

		log.Info(ctx, nil, "Updating OIDC settings")
		c.UpdateAccessPolicy(ctx, cfg)
	} else {
		log.Info(ctx, nil, "Deleting OIDC settings")
		c.DeleteAccessPolicy(ctx, ap.Name)
	}

	return nil
}
