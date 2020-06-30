package envoyfilter

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"istio-keycloak/api"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	scope := log.WithField("EnvoyFilter", fmt.Sprintf("%s/%s", req.Namespace, req.Name))
	ctx = log.NewContext(ctx, scope)

	ef := istionetworking.EnvoyFilter{}
	err := c.Get(ctx, req.NamespacedName, &ef)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting EnvoyFilter")
	}

	dup, err := c.isDuplicate(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, err
	} else if dup {
		log.FromContext(ctx).Info("Deleting duplicate")
		err = c.Delete(ctx, &ef)
		return reconcile.Result{}, errors.Wrap(err, "failed deleting EnvoyFilter")
	}

	aps, err := c.fetchAccessPolicies(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, ap := range aps {
		log.FromContext(ctx).WithField("AccessPolicy", ap.Name).
			Info("Including AccessPolicy in EnvoyFilter")
	}

	err = newEnvoyFilter(&ef, aps)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.FromContext(ctx).Info("Updating resource")
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
			reflect.DeepEqual(allEfs.Items[i].Spec.GetWorkloadSelector().GetLabels(), ef.Spec.GetWorkloadSelector().GetLabels()) {
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
	allAps := api.AccessPolicyList{}
	err := c.List(ctx, &allAps)
	if err != nil {
		return nil, errors.Wrap(err, "failed listing AccessPolicies")
	}

	aps := make([]*state.AccessPolicy, 0, len(allAps.Items))
	for i := range allAps.Items {
		if reflect.DeepEqual(allAps.Items[i].Status.GetIngress().GetSelector(), ef.Spec.GetWorkloadSelector().GetLabels()) {
			ap, err := state.NewAccessPolicy(&allAps.Items[i], nil)
			if err != nil {
				log.FromContext(ctx).WithError(err).WithField("AccessPolicy", allAps.Items[i].Name).
					Error("Invalid AccessPolicy")
			} else {
				aps = append(aps, ap)
			}
		}
	}

	return aps, nil
}
