package accesspolicy

import (
	"istio-keycloak/api"
	"istio-keycloak/config"
	istionetworkingapi "istio.io/api/networking/v1alpha3"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func newEnvoyFilter(ap *api.AccessPolicy) *istionetworking.EnvoyFilter {
	ef := &istionetworking.EnvoyFilter{}
	ef.Namespace = config.Controller.IstioRootNamespace
	ef.GenerateName = config.Controller.EnvoyFilterNamePrefix
	ef.Spec.WorkloadSelector = &istionetworkingapi.WorkloadSelector{}
	ef.Spec.WorkloadSelector.Labels = ap.Status.Ingress.Selector
	return ef
}
