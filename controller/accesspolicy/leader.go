package accesspolicy

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizer = "finalizer.istio-oidc"

type leaderReconciler struct {
	client.Client
	record.EventRecorder
}

func (r *leaderReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	ctx = log.WithValues(ctx, "AccessPolicy", req.Namespace+"/"+req.Name, "leader", "true")

	ap := api.AccessPolicy{}
	err := r.Get(ctx, req.NamespacedName, &ap)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.Wrap(err, "failed getting AccessPolicy")
	}

	if ap.DeletionTimestamp.IsZero() && !contains(ap.Finalizers, finalizer) {
		log.Info(ctx, nil, "Adding finalizer")
		ap.Finalizers = append(ap.Finalizers, finalizer)
		if err := r.Update(ctx, &ap); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed adding AccessPolicy finalizer")
		}
	}

	err = r.reconcileStatus(ctx, &ap)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileEnvoyFilter(ctx, &ap)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !ap.DeletionTimestamp.IsZero() && contains(ap.Finalizers, finalizer) {
		log.Info(ctx, nil, "Removing finalizer")
		ap.Finalizers = remove(ap.Finalizers, finalizer)
		if err := r.Update(ctx, &ap); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "failed removing AccessPolicy finalizer")
		}
	}

	return reconcile.Result{}, nil
}

func (r *leaderReconciler) reconcileStatus(ctx context.Context, ap *api.AccessPolicy) error {
	if ap.DeletionTimestamp.IsZero() {
		gwName := ap.Spec.Gateway
		gwKey := types.NamespacedName{Namespace: ap.Namespace, Name: gwName}
		gw := istionetworking.Gateway{}
		err := r.Get(ctx, gwKey, &gw)
		if err != nil {
			log.Error(ctx, err, "Failed getting Gateway")
			r.Event(ap, "Warning", "MissingGateway", "Failed getting gateway")
			return nil
		}

		ap.Status.Ingress.Selector = selector(&gw)
		ap.Status.VirtualHosts = virtualHosts(&gw)

		log.Info(ctx, nil, "Updating status")
		err = r.Status().Update(ctx, ap)
		if err != nil {
			err = errors.Wrap(err, "failed updating AccessPolicy status")
			return err
		}
	}

	return nil
}

func (r *leaderReconciler) reconcileEnvoyFilter(ctx context.Context, ap *api.AccessPolicy) error {
	if len(ap.Status.GetIngress().GetSelector()) == 0 {
		log.Error(ctx, nil, "Missing workload selector")
		return nil
	}

	allEfs := istionetworking.EnvoyFilterList{}
	err := r.List(ctx, &allEfs,
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
		err = r.Create(ctx, newEnvoyFilter(ap))
		if err != nil {
			return errors.Wrap(err, "failed creating EnvoyFilter")
		}
	} else {
		vals := log.MakeValues("EnvoyFilter", efs[0].Name)
		log.Info(ctx, vals, "Found EnvoyFilter")
	}

	return nil
}
