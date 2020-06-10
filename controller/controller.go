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
	"time"
)

// TODO: Configurable
var (
	IstioRootNamespace    = "istio-system"
	EnvoyFilterNamePrefix = "ext-authz-"
	ExtAuthzClusterName   string
	ExtAuthzTimeout       time.Duration
)

type Controller struct {
	client.Client
	credfilt *eventFilter
	gwfilt   *eventFilter
	effilt   *eventFilter
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

	c.credfilt = newEventFilter(&core.Secret{})
	c.gwfilt = newEventFilter(&istionetworking.Gateway{})
	c.effilt = newEventFilter(&istionetworking.EnvoyFilter{})

	err = builder.
		ControllerManagedBy(mgr).
		WithEventFilter(&predicate.ResourceVersionChangedPredicate{}).
		WithEventFilter(c.credfilt).
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

	pols, err := fetchAccessPolicies(ctx, c)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, ap := range pols {
		if ap.AccessPolicy == nil {
			log.WithField("AccessPolicy", ap.key).Error("Skipping reconciliation")
			partial = true
		} else {
			c.credfilt.track(ap.credentials)
			c.gwfilt.track(ap.gateway)
		}
	}

	for _, ingress := range ingresses(pols) {
		err := c.reconcileEnvoyFilter(ctx, *ingress, pols)
		if err != nil {
			log.WithError(err).Error("Unable to reconcile")
			partial = true
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

func (c *Controller) reconcileEnvoyFilter(ctx context.Context, i ingress, pols []accessPolicy) error {
	scope := log.WithField("EnvoyFilter", i.String())
	scope.Info("Reconciling")

	next, err := newEnvoyFilter(i, pols)
	if err != nil {
		return errors.Wrap(err, "unable to construct next EnvoyFilter")
	}

	curr, err := fetchEnvoyFilter(ctx, c, &i)
	if err != nil {
		key := i.String()
		return errors.Wrap(err, "unable to fetch EnvoyFilter", "EnvoyFilter", key)
	}

	if curr == nil {
		curr = next
		err = c.Create(ctx, curr)
		if err != nil {
			key := fmt.Sprintf("%s/%s", curr.Namespace, curr.Name)
			return errors.Wrap(err, "unable to create EnvoyFilter", "EnvoyFilter", key)
		}
	} else if reflect.DeepEqual(curr.Spec, next.Spec) {
		scope.Info("Already in desired state")
	} else {
		curr.Spec = next.Spec
		err = c.Update(ctx, curr)
		if err != nil {
			key := fmt.Sprintf("%s/%s", curr.Namespace, curr.Name)
			return errors.Wrap(err, "unable to update EnvoyFilter", "EnvoyFilter", key)
		}
	}

	c.effilt.track(curr.GetUID())
	scope.Info("Finished reconciliation")
	return nil
}
