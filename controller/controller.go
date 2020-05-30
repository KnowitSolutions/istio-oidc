package controller

import (
	"github.com/apex/log"
	"istio-keycloak/api/v1"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// TODO: Attach manager's log to Apex log
type Controller struct {
	client.Client
}

func (r *Controller) SetupWithManager(mgr ctrl.Manager) {
	scheme := mgr.GetScheme()

	err := istionetworking.AddToScheme(scheme)
	if err != nil {
		log.WithError(err).Fatal("Unable to add Istio networking to scheme")
	}

	err = api.AddToScheme(scheme)
	if err != nil {
		log.WithError(err).Fatal("Unable to add CRD to scheme")
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&api.AccessPolicy{}).
		Watches(
			&source.Kind{Type: &istionetworking.Gateway{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: &GatewayMapper{}}).
		Owns(&istionetworking.EnvoyFilter{}).
		Complete(&Controller{})

	if err != nil {
		log.WithError(err).Fatal("Unable to create controller")
	}
}

func (r *Controller) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	log.WithFields(log.Fields{
		"namespace": req.Namespace,
		"name": req.Name,
	}).Info("Starting reconciliation")
	return reconcile.Result{}, nil
}

