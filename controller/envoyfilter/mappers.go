package envoyfilter

import (
	"context"
	"github.com/apex/log"
	"istio-keycloak/api"
	"istio-keycloak/config"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type mapper struct {
	client.Client
}

func (m *mapper) Map(obj handler.MapObject) []reconcile.Request {
	ctx := context.Background()
	ap := obj.Object.(*api.AccessPolicy)

	efs := istionetworking.EnvoyFilterList{}
	err := m.List(ctx, &efs,
		client.InNamespace(config.Controller.IstioRootNamespace),
		client.MatchingLabels(config.Controller.EnvoyFilterLabels))
	if err != nil {
		log.WithError(err).Error("Failed fetching EnvoyFilters")
		return nil
	}

	reqs := make([]reconcile.Request, 0, len(efs.Items))
	for _, ef := range efs.Items {
		if reflect.DeepEqual(ap.Status.GetIngress().GetSelector(), ef.Spec.GetWorkloadSelector().GetLabels()) {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: ef.Namespace,
				Name:      ef.Name,
			}})
		}
	}

	return reqs
}
