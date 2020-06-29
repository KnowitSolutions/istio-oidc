package accesspolicy

import (
	"context"
	"github.com/apex/log"
	"istio-keycloak/api"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type mapper struct {
	client.Client
}

func (m *mapper) Map(obj handler.MapObject) []reconcile.Request {
	ctx := context.Background()
	name := obj.Meta.GetName()
	ns := obj.Meta.GetNamespace()

	aps := api.AccessPolicyList{}
	err := m.List(ctx, &aps, client.InNamespace(ns))
	if err != nil {
		log.WithError(err).WithField("namespace", ns).
			Error("Failed fetching AccessPolicies")
		return nil
	}

	reqs := make([]reconcile.Request, 0, len(aps.Items))
	for _, ap := range aps.Items {
		if ap.Spec.Gateway == name {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: ns,
				Name:      ap.Name,
			}})
		}
	}

	return reqs
}
