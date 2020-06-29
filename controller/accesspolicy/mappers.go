package accesspolicy

import (
	"context"
	"github.com/apex/log"
	"istio-keycloak/api"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type mapper struct {
	client.Client
	namespaced bool
	isRelated  func(*handler.MapObject, *api.AccessPolicy) bool
}

func (m *mapper) Map(obj handler.MapObject) []reconcile.Request {
	ctx := context.Background()
	ns := obj.Meta.GetNamespace()

	var opts []client.ListOption
	if m.namespaced {
		opts = []client.ListOption{client.InNamespace(ns)}
	}

	aps := api.AccessPolicyList{}
	err := m.List(ctx, &aps, opts...)
	if err != nil {
		log.WithError(err).WithField("namespace", ns).
			Error("Failed fetching AccessPolicies")
		return nil
	}

	reqs := make([]reconcile.Request, 0, len(aps.Items))
	for _, ap := range aps.Items {
		if m.isRelated(&obj, &ap) {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: ap.Namespace,
				Name:      ap.Name,
			}})
		}
	}

	return reqs
}

func newGatewayMapper(mgr ctrl.Manager) handler.Mapper {
	return &mapper{mgr.GetClient(), true, gwIsRelated}
}

func gwIsRelated(obj *handler.MapObject, ap *api.AccessPolicy) bool {
	return ap.Spec.Gateway == obj.Meta.GetName()
}

func newSecretMapper(mgr ctrl.Manager) handler.Mapper {
	return &mapper{mgr.GetClient(), true, secretIsRelated}
}

func secretIsRelated(obj *handler.MapObject, ap *api.AccessPolicy) bool {
	return ap.Spec.OIDC.CredentialsSecret.Name == obj.Meta.GetName()
}

func newEnvoyFilterMapper(mgr ctrl.Manager) handler.Mapper {
	return &mapper{mgr.GetClient(), false, efIsRelated}
}

func efIsRelated(obj *handler.MapObject, ap *api.AccessPolicy) bool {
	apSel := ap.Status.GetIngress().GetSelector()
	efSel := obj.Object.(*istionetworking.EnvoyFilter).Spec.GetWorkloadSelector().GetLabels()
	return reflect.DeepEqual(apSel, efSel)
}
