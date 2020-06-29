package envoyfilter

import (
	"context"
	"github.com/apex/log"
	"istio-keycloak/api"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
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
		return reconcile.Result{}, errors.Wrap(err, "failed getting EnvoyFilter")
	}

	dup, err := c.isDuplicate(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, err
	} else if dup {
		log.WithField("EnvoyFilter", ef.Name).Info("Deleting duplicate")
		err = c.Delete(ctx, &ef)
		return reconcile.Result{}, errors.Wrap(err, "failed deleting EnvoyFilter")
	}

	aps, err := c.fetchAccessPolicies(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = mkEnvoyFilter(&ef, aps)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.WithField("EnvoyFilter", ef.Name).Info("Updating")
	err = c.Update(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed updating EnvoyFilter")
	}

	return reconcile.Result{}, nil
}

func (c *controller) isDuplicate(ctx context.Context, ef *istionetworking.EnvoyFilter) (bool, error) {
	allEfs := istionetworking.EnvoyFilterList{}
	err := c.List(ctx, &allEfs, client.InNamespace(config.Controller.IstioRootNamespace))
	if err != nil {
		return false, errors.Wrap(err, "failed listing EnvoyFilters")
	}

	efs := make([]*istionetworking.EnvoyFilter, 0, len(allEfs.Items))
	for i := range allEfs.Items {
		if strings.HasPrefix(allEfs.Items[i].Name, config.Controller.EnvoyFilterNamePrefix) &&
			reflect.DeepEqual(allEfs.Items[i].Spec.WorkloadSelector, ef.Spec.WorkloadSelector) {
			efs = append(efs, &allEfs.Items[i])
		}
	}

	sort.Slice(efs, func(i, j int) bool {
		iTime := efs[i].CreationTimestamp
		jTime := efs[j].CreationTimestamp
		return iTime.Before(&jTime)
	})

	return ef.Name != efs[0].Name, nil
}

func (c *controller) fetchAccessPolicies(ctx context.Context, ef *istionetworking.EnvoyFilter) ([]*state.AccessPolicy, error) {
	owners := make(map[types.UID]bool, len(ef.OwnerReferences))
	for _, owner := range ef.OwnerReferences {
		owners[owner.UID] = true
	}

	allAps := api.AccessPolicyList{}
	err := c.List(ctx, &allAps)
	if err != nil {
		return nil, errors.Wrap(err, "failed listing AccessPolicies")
	}

	aps := make([]*state.AccessPolicy, 0, len(allAps.Items))
	for i := range allAps.Items {
		if owners[allAps.Items[i].UID] {
			ap, err := state.NewAccessPolicy(&allAps.Items[i], nil)
			if err != nil {
				log.WithError(err).WithField("AccessPolicy", allAps.Items[i].Name).
					Error("Invalid AccessPolicy")
			} else {
				aps = append(aps, ap)
			}
		}
	}

	return aps, nil
}
