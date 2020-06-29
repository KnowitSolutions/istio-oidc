package accesspolicy

import (
	"context"
	"istio-keycloak/api/v1"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
	istionetworkingapi "istio.io/api/networking/v1alpha3"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
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
	state.OidcCommunicatorStore
}

func (c *controller) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	ap := api.AccessPolicy{}
	err := c.Get(ctx, req.NamespacedName, &ap)
	if err != nil {
		return reconcile.Result{}, err
	}

	if ap.DeletionTimestamp.IsZero() && !contains(ap.Finalizers, finalizer) {
		ap.Finalizers = append(ap.Finalizers, finalizer)
		if err := c.Update(ctx, &ap); err != nil {
			return reconcile.Result{}, err
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
		ap.Finalizers = remove(ap.Finalizers, finalizer)
		if err := c.Update(ctx, &ap); err != nil {
			return reconcile.Result{}, err
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
			return err
		}

		ap.Status.Ingress.Selector = selector(&gw)
		ap.Status.VirtualHosts = virtualHosts(&gw)

		err = c.Status().Update(ctx, ap)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *controller) reconcileEnvoyFilter(ctx context.Context, ap *api.AccessPolicy) error {
	// TODO: Verify that AP is good to go

	efs := istionetworking.EnvoyFilterList{}
	err := c.List(ctx, &efs)
	if err != nil {
		return err
	}

	filtered := make([]*istionetworking.EnvoyFilter, 0, len(efs.Items))
	for i := range efs.Items {
		if strings.HasPrefix(efs.Items[i].Name, config.Controller.EnvoyFilterNamePrefix) &&
			reflect.DeepEqual(efs.Items[i].Spec.WorkloadSelector, ap.Status.Ingress.Selector) {
			filtered = append(filtered, &efs.Items[i])
		}
	}

	if len(filtered) == 0 {
		ef := istionetworking.EnvoyFilter{}
		ef.Namespace = config.Controller.IstioRootNamespace
		ef.GenerateName = config.Controller.EnvoyFilterNamePrefix
		ef.Spec.WorkloadSelector = &istionetworkingapi.WorkloadSelector{}
		ef.Spec.WorkloadSelector.Labels = ap.Status.Ingress.Selector
		err = c.Create(ctx, &ef)
		if err != nil {
			return err
		}
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
			return err
		}

		cfg, err := state.NewAccessPolicy(ap, &cred)
		if err != nil {
			return err
		}

		c.UpdateOicd(ctx, cfg)
	} else {
		c.DeleteOidc(ap.Name)
	}

	return nil
}
