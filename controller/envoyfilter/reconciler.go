package envoyfilter

import (
	"context"
	"istio-keycloak/api"
	"istio-keycloak/config"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"istio-keycloak/state/accesspolicy"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sort"
)

type reconciler struct {
	client.Client
}

func (r *reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	ctx = log.WithValues(ctx, "EnvoyFilter", req.Namespace+"/"+req.Name)

	ef := istionetworking.EnvoyFilter{}
	err := r.Get(ctx, req.NamespacedName, &ef)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting EnvoyFilter")
	}

	dup, err := r.isDuplicate(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, err
	} else if dup {
		log.Info(ctx, nil, "Deleting duplicate")
		err = r.Delete(ctx, &ef)
		return reconcile.Result{}, errors.Wrap(err, "failed deleting EnvoyFilter")
	}

	aps, err := r.fetchAccessPolicies(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, err
	}

	if len(aps) == 0 {
		log.Info(ctx, nil, "Deleting resource")
		err = r.Delete(ctx, &ef)
		return reconcile.Result{}, err
	}

	for _, ap := range aps {
		vals := log.MakeValues("AccessPolicy", ap.Name)
		log.Info(ctx, vals, "Including AccessPolicy in EnvoyFilter")
	}

	newEnvoyFilter(&ef, aps)
	log.Info(ctx, nil, "Updating resource")
	err = r.Update(ctx, &ef)
	if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed updating EnvoyFilter")
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) isDuplicate(ctx context.Context, ef *istionetworking.EnvoyFilter) (bool, error) {
	allEfs := istionetworking.EnvoyFilterList{}
	err := r.List(ctx, &allEfs,
		client.InNamespace(config.Controller.IstioRootNamespace),
		client.MatchingLabels(config.Controller.EnvoyFilterLabels))
	if err != nil {
		return false, errors.Wrap(err, "failed listing EnvoyFilters")
	}

	efs := make([]*istionetworking.EnvoyFilter, 0, len(allEfs.Items))
	for i := range allEfs.Items {
		if reflect.DeepEqual(allEfs.Items[i].Spec.GetWorkloadSelector().GetLabels(), ef.Spec.GetWorkloadSelector().GetLabels()) {
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

func (r *reconciler) fetchAccessPolicies(ctx context.Context, ef *istionetworking.EnvoyFilter) ([]*accesspolicy.AccessPolicy, error) {
	allAps := api.AccessPolicyList{}
	err := r.List(ctx, &allAps)
	if err != nil {
		return nil, errors.Wrap(err, "failed listing AccessPolicies")
	}

	aps := make([]*accesspolicy.AccessPolicy, 0, len(allAps.Items))
	for i := range allAps.Items {
		if reflect.DeepEqual(allAps.Items[i].Status.GetIngress().GetSelector(), ef.Spec.GetWorkloadSelector().GetLabels()) {
			ap, err := accesspolicy.NewAccessPolicy(&allAps.Items[i], nil)
			if err != nil {
				err = errors.Wrap(err, "", "AccessPolicy", allAps.Items[i].Name)
				log.Error(ctx, err, "Invalid AccessPolicy")
			} else {
				aps = append(aps, ap)
			}
		}
	}

	return aps, nil
}
