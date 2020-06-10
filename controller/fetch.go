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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func fetchAccessPolicies(ctx context.Context, c client.Client) ([]accessPolicy, error) {
	log.Info("Fetching AccessPolicies")

	aps := api.AccessPolicyList{}
	err := c.List(ctx, &aps)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch AccessPolicies")
	}

	pols := make([]accessPolicy, len(aps.Items))
	for i, ap := range aps.Items {
		scope := log.WithField("AccessPolicy", fmt.Sprintf("%s/%s", ap.Namespace, ap.Name))

		scope.Info("Fetching credentials Secret")
		cred := core.Secret{}
		err := c.Get(ctx, credentialsKey(&ap), &cred)
		if err != nil {
			scope.WithError(err).Error("Unable to fetch credentials")
			continue
		}

		scope.Info("Fetching Gateway")
		gw := istionetworking.Gateway{}
		err = c.Get(ctx, gatewayKey(&ap), &gw)
		if err != nil {
			scope.WithError(err).Error("Unable to fetch gateway")
			continue
		}

		pols[i] = *newAccessPolicy(&ap, &cred, &gw)
	}

	return pols, nil
}

func fetchEnvoyFilter(ctx context.Context, c client.Client, i *ingress) (*istionetworking.EnvoyFilter, error) {
	scope := log.WithField("EnvoyFilter", i.String())
	scope.Info("Fetching EnvoyFilter")

	efs := istionetworking.EnvoyFilterList{}
	err := c.List(ctx, &efs, client.InNamespace(i.namespace))
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch EnvoyFilters")
	}

	var ef *istionetworking.EnvoyFilter
	for _, elem := range efs.Items {
		if ef != nil {
			id := fmt.Sprintf("%s/%s", elem.Namespace, elem.Name)
			scope.WithField("EnvoyFilter", id).Info("Deleting duplicate EnvoyFilter")

			err = c.Delete(ctx, &elem)
			if err != nil {
				scope.WithError(err).Error("Unable to delete duplicate")
			}

			continue
		}

		if strings.HasPrefix(elem.Name, EnvoyFilterNamePrefix) &&
			reflect.DeepEqual(elem.Spec.GetWorkloadSelector().GetLabels(), i.selector) {
			ef = &elem
		}
	}

	return ef, nil
}
