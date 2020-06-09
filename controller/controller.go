package controller

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"istio-keycloak/api/v1"
	"istio-keycloak/logging/errors"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strings"
)

type Controller struct {
	client.Client
	gwfilt *eventFilter
	effilt *eventFilter
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager) {
	scheme := mgr.GetScheme()
	err := core.AddToScheme(scheme)
	if err != nil {
		log.WithError(err).Fatal("Unable to add Kubernetes core to scheme")
	}
	err = istionetworking.AddToScheme(scheme)
	if err != nil {
		log.WithError(err).Fatal("Unable to add Istio networking to scheme")
	}
	err = api.AddToScheme(scheme)
	if err != nil {
		log.WithError(err).Fatal("Unable to add CRD to scheme")
	}

	c.gwfilt = newEventFilter(&istionetworking.Gateway{})
	c.effilt = newEventFilter(&istionetworking.EnvoyFilter{})

	err = builder.
		ControllerManagedBy(mgr).
		WithEventFilter(&predicate.ResourceVersionChangedPredicate{}).
		WithEventFilter(c.gwfilt).
		WithEventFilter(c.effilt).
		For(&api.AccessPolicy{}).
		Watches(&source.Kind{Type: &istionetworking.Gateway{}}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{Type: &istionetworking.EnvoyFilter{}}, &handler.EnqueueRequestForObject{}).
		Complete(c)
	if err != nil {
		log.WithError(err).Fatal("Unable to make controller")
	}
}

func (c *Controller) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	log.Info("Starting reconciliation")
	ctx := context.Background()
	partial := false

	log.Info("Collecting AccessPolicies")
	aps, err := c.getAccessPolicies(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	svcs := make([]service, 0, len(aps))
	for _, ap := range aps {
		scope := log.WithField("AccessPolicy", fmt.Sprintf("%s/%s", ap.Namespace, ap.Name))
		scope.Info("Collecting dependencies")
		svc, err := c.collectService(ctx, &ap)
		if err != nil {
			scope.WithError(err).Error("Skipping reconciliation")
			partial = true
		} else {
			svcs = append(svcs, svc)
		}
	}

	for _, ingress := range ingresses(svcs) {
		scope := log.WithField("EnvoyFilter", ingress.String())
		scope.Info("Reconciling")
		updated, err := c.reconcileEnvoyFilter(ctx, *ingress, svcs)
		if err != nil {
			scope.WithError(err).Error("Unable to reconcile")
			partial = true
		} else if updated {
			scope.Info("Reconciled")
		} else {
			scope.Info("Already in desired state")
		}
	}

	if partial {
		return reconcile.Result{}, errors.New("partial reconciliation")
	} else {
		c.gwfilt.clean()
		c.effilt.clean()
		return reconcile.Result{}, nil
	}
}

func (c *Controller) getAccessPolicies(ctx context.Context) ([]api.AccessPolicy, error) {
	aps := api.AccessPolicyList{}
	err := c.List(ctx, &aps)
	return aps.Items, errors.Wrap(err, "unable to fetch AccessPolicies")
}

func (c *Controller) collectService(ctx context.Context, ap *api.AccessPolicy) (service, error) {
	cred := core.Secret{}
	err := c.Get(ctx, ap.CredentialsKey(), &cred)
	if err != nil {
		return service{}, errors.Wrap(err, "unable to fetch credentials")
	}

	gw := istionetworking.Gateway{}
	err = c.Get(ctx, ap.GatewayKey(), &gw)
	if err != nil {
		return service{}, errors.Wrap(err, "unable to fetch gateway")
	}
	c.gwfilt.track(&gw)

	return service{
		Service: ap.ToConfig(&cred),
		ingress: *newIngress(&gw),
		vhosts:  virtualHosts(&gw),
	}, nil
}

func ingresses(svcs []service) []*ingress {
	hash := make(map[string]*ingress, len(svcs))
	for _, svc := range svcs {
		hash[svc.ingress.key] = &svc.ingress
	}

	i := 0
	list := make([]*ingress, len(hash))
	for _, ingress := range hash {
		list[i] = ingress
		i++
	}

	return list
}

func (c *Controller) reconcileEnvoyFilter(ctx context.Context, ingress ingress, svcs []service) (bool, error) {
	next, err := mkEnvoyFilter(ingress, svcs)
	if err != nil {
		return true, errors.Wrap(err, "unable to construct next EnvoyFilter")
	}

	list := istionetworking.EnvoyFilterList{}
	err = c.List(ctx, &list, client.InNamespace(ingress.namespace))
	if err != nil {
		return true, errors.Wrap(err, "unable to fetch EnvoyFilters")
	}

	var curr *istionetworking.EnvoyFilter
	for _, ef := range list.Items {
		if !strings.HasPrefix(ef.Name, EnvoyFilterName) {
			continue
		} else if !reflect.DeepEqual(ef.Spec.GetWorkloadSelector().GetLabels(), ingress.selector) {
			continue
		} else if curr != nil {
			id := fmt.Sprintf("%s/%s", ef.Namespace, ef.Name)
			log.WithField("EnvoyFilter", id).Info("Deleting duplicate EnvoyFilter")

			err = c.Delete(ctx, &ef)
			if err != nil {
				return true, errors.Wrap(err, "unable to delete duplicate", "EnvoyFilter", id)
			}
		} else {
			curr = &ef
		}
	}

	if curr == nil {
		err = c.Create(ctx, next)
		c.effilt.track(next)
		return true, errors.Wrap(err, "unable to create EnvoyFilter")
	} else if !reflect.DeepEqual(curr.Spec, next.Spec) {
		curr.Spec = next.Spec
		err = c.Update(ctx, curr)
		c.effilt.track(curr)

		id := fmt.Sprintf("%s/%s", curr.Namespace, curr.Name)
		return true, errors.Wrap(err, "unable to update EnvoyFilter", "EnvoyFilter", id)
	} else {
		c.effilt.track(curr)
		return false, nil
	}
}
