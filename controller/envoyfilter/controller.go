package envoyfilter

import (
	"context"
	"istio-keycloak/api/v1"
	"istio-keycloak/config"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sort"
	"strings"
)

type controller struct {
	client.Client
}

func (c *controller) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	ef := istionetworking.EnvoyFilter{}
	err := c.Get(ctx, req.NamespacedName, &ef)
	if err != nil {
		return reconcile.Result{}, err
	}

	allEfs := istionetworking.EnvoyFilterList{}
	err = c.List(ctx, &allEfs, client.InNamespace(config.Controller.IstioRootNamespace))
	if err != nil {
		return reconcile.Result{}, err
	}

	efs := make([]*istionetworking.EnvoyFilter, 0, len(allEfs.Items))
	for i := range allEfs.Items {
		if strings.HasPrefix(allEfs.Items[i].Name, config.Controller.EnvoyFilterNamePrefix) &&
			reflect.DeepEqual(allEfs.Items[i].Spec.WorkloadSelector, ef.Spec.WorkloadSelector) {
			efs = append(efs, &allEfs.Items[i])
		}
	}

	sort.Slice(efs, func(i, j int) bool {
		return strings.Compare(efs[i].Name, efs[j].Name) == -1
	})

	if ef.Name != efs[0].Name {
		err = c.Delete(ctx, &ef)
		return reconcile.Result{}, err
	}

	allAps := api.AccessPolicyList{}
	err = c.List(ctx, &allEfs)
	if err != nil {
		return reconcile.Result{}, err
	}

	aps := make([]*state.AccessPolicy, 0, len(allAps.Items))
	for i := range allAps.Items {
		if reflect.DeepEqual(allAps.Items[i].Status.Ingress.Selector, ef.Spec.WorkloadSelector) {
			ap, err := state.NewAccessPolicy(&allAps.Items[i], nil)
			if err != nil {
				aps = append(aps, ap)
			}
		}
	}
	
	err = mkEnvoyFilter(&ef, aps)
	if err != nil {
		return reconcile.Result{}, err
	}
	
	err = c.Update(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
