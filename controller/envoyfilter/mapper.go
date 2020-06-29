package envoyfilter

import (
	"context"
	"github.com/apex/log"
	"istio-keycloak/api/v1"
	"istio-keycloak/config"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

type mapper struct {
	client.Client
}

func (m *mapper) Map(obj handler.MapObject) []reconcile.Request {
	ctx := context.Background()
	ap := obj.Object.(*api.AccessPolicy)

	efs := istionetworking.EnvoyFilterList{}
	err := m.List(ctx, &efs, client.InNamespace(config.Controller.IstioRootNamespace))
	if err != nil {
		log.WithError(err).WithField("namespace", config.Controller.IstioRootNamespace).
			Error("Failed fetching EnvoyFilters")
		return nil
	}

	reqs := make([]reconcile.Request, 0, len(efs.Items))
	for i := range efs.Items {
		if strings.HasPrefix(efs.Items[i].Name, config.Controller.EnvoyFilterNamePrefix) &&
			reflect.DeepEqual(efs.Items[i].Spec.WorkloadSelector, ap.Status.Ingress.Selector) {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: efs.Items[i].Namespace,
				Name:      efs.Items[i].Name,
			}})
		}
	}

	return reqs
}
