package accesspolicy

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"istio-keycloak/api"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"sync"
)

const finalizer = "finalizer.istio-keycloak"

type controller struct {
	client.Client
	*runtime.Scheme
	state.OidcCommunicatorStore
}

func (c *controller) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	scope := log.WithField("AccessPolicy", fmt.Sprintf("%s/%s", req.Namespace, req.Name))
	ctx = log.NewContext(ctx, scope)

	ap := api.AccessPolicy{}
	err := c.Get(ctx, req.NamespacedName, &ap)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting AccessPolicy")
	}

	if ap.DeletionTimestamp.IsZero() && !contains(ap.Finalizers, finalizer) {
		log.FromContext(ctx).Info("Adding finalizer")
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
		log.FromContext(ctx).Info("Removing finalizer")
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

		log.FromContext(ctx).Info("Updating status")
		err = c.Status().Update(ctx, ap)
		if err != nil {
			return errors.Wrap(err, "failed updating AccessPolicy status")
		}
	}

	return nil
}

func (c *controller) reconcileEnvoyFilter(ctx context.Context,  ap *api.AccessPolicy) error {
	if len(ap.Status.GetIngress().GetSelector()) == 0 {
		log.FromContext(ctx).Info("Missing workload selector")
		return nil
	}

	allEfs := istionetworking.EnvoyFilterList{}
	err := c.List(ctx, &allEfs, client.InNamespace(config.Controller.IstioRootNamespace))
	if err != nil {
		return errors.Wrap(err, "failed listing EnvoyFilters")
	}

	efs := make([]*istionetworking.EnvoyFilter, 0, len(allEfs.Items))
	for i := range allEfs.Items {
		if strings.HasPrefix(allEfs.Items[i].Name, config.Controller.EnvoyFilterNamePrefix) &&
			reflect.DeepEqual(allEfs.Items[i].Spec.GetWorkloadSelector().GetLabels(), ap.Status.Ingress.Selector) {
			efs = append(efs, &allEfs.Items[i])
		}
	}

	if len(efs) == 0 {
		log.FromContext(ctx).Info("Creating EnvoyFilter")
		err = c.Create(ctx, newEnvoyFilter(ap))
		if err != nil {
			return errors.Wrap(err, "failed creating EnvoyFilter")
		}
	} else {
		log.FromContext(ctx).WithField("EnvoyFilter", efs[0].Name).
			Info("Found EnvoyFilter")
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

		cfg, err := state.NewAccessPolicy(ap, &cred)
		if err != nil {
			return errors.Wrap(err, "invalid AccessPolicy")
		}

		log.FromContext(ctx).Info("Updating OIDC settings")
		c.UpdateOicd(ctx, cfg)
	} else {
		log.FromContext(ctx).Info("Deleting OIDC settings")
		c.DeleteOidc(ap.Name)
	}

	return nil
}
