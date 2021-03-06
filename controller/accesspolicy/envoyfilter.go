package accesspolicy

import (
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/config"
	istionetworkingapi "istio.io/api/networking/v1alpha3"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func newEnvoyFilter(ap *api.AccessPolicy) *istionetworking.EnvoyFilter {
	ef := &istionetworking.EnvoyFilter{}
	ef.Namespace = config.Controller.IstioRootNamespace
	ef.Labels = config.Controller.EnvoyFilterLabels
	ef.GenerateName = config.Controller.EnvoyFilterNamePrefix
	ef.Spec.WorkloadSelector = &istionetworkingapi.WorkloadSelector{}
	ef.Spec.WorkloadSelector.Labels = ap.Status.Ingress.Selector
	return ef
}
