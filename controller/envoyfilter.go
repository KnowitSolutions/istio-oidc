package controller

import (
	"fmt"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
	istionetworkingapi "istio.io/api/networking/v1alpha3"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func newEnvoyFilter(ingress ingress, pols []accessPolicy) (*istionetworking.EnvoyFilter, error) {
	count := 2
	for _, pol := range pols {
		if pol.ingress.key == ingress.key {
			count += len(pol.vhosts) * len(pol.Routes)
		}
	}

	ef := &istionetworking.EnvoyFilter{}
	ef.Namespace = IstioRootNamespace
	ef.GenerateName = EnvoyFilterNamePrefix
	ef.Spec.WorkloadSelector = &istionetworkingapi.WorkloadSelector{}
	ef.Spec.WorkloadSelector.Labels = ingress.selector
	ef.Spec.ConfigPatches = make([]*istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch, 0, count)

	extAuthzFilter := &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch{}
	applyToHttpFilter(extAuthzFilter)
	matchEnvoyRouter(extAuthzFilter)
	insertBefore(extAuthzFilter)
	extAuthz(extAuthzFilter)
	ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, extAuthzFilter)

	extAuthzDisable := &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch{}
	applyToVirtualHost(extAuthzDisable)
	matchGateway(extAuthzDisable)
	merge(extAuthzDisable)
	_ = extAuthzPerRoute(extAuthzDisable, "", nil)
	ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, extAuthzDisable)

	for _, pol := range pols {
		if pol.ingress.key != ingress.key {
			continue
		}

		for _, vhost := range pol.vhosts {
			patch := &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch{}
			applyToVirtualHost(patch)
			matchVirtualHost(patch, vhost)
			merge(patch)
			err := extAuthzPerRoute(patch, pol.Name, &pol.Global)
			if err != nil {
				return nil, err
			}

			ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, patch)

			for route, routeData := range pol.Routes {
				patch := &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch{}
				applyToHttpRoute(patch)
				matchVirtualHostRoute(patch, vhost, route)
				merge(patch)
				err = extAuthzPerRoute(patch, pol.Name, &routeData)
				if err != nil {
					return nil, err
				}

				ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, patch)
			}
		}
	}

	return ef, nil
}

func extAuthz(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.Patch.Value = newStruct(map[string]interface{}{
		"name": "envoy.filters.http.ext_authz",
		"typed_config": map[string]interface{}{
			"@type": "type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthz",
			"grpc_service": map[string]interface{}{
				"envoy_grpc": map[string]interface{}{
					"cluster_name": ExtAuthzClusterName,
				},
				"timeout": fmt.Sprintf("%.fs", ExtAuthzTimeout.Seconds()),
			},
		},
	})
}

func extAuthzPerRoute(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch, policy string, route *config.Route) error {
	cfg := map[string]interface{}{
		"@type": "type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthzPerRoute",
	}

	if route != nil && route.EnableAuthz {
		roles, err := route.Roles.Encode()
		if err != nil {
			return errors.Wrap(err, "failed to construct EnvoyFilter")
		}

		exts := make(map[string]interface{}, 2)
		if len(route.Roles) > 0 {
			exts = map[string]interface{}{
				config.AccessPolicyKey: policy,
				config.RolesKey:        roles,
			}
		}

		cfg["check_settings"] = map[string]interface{}{"context_extensions": exts}
	} else {
		cfg["disabled"] = true
	}

	patch.Patch.Value = newStruct(map[string]interface{}{
		"typed_per_filter_config": map[string]interface{}{
			"envoy.filters.http.ext_authz": cfg,
		},
	})

	return nil
}

func applyToHttpFilter(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.ApplyTo = istionetworkingapi.EnvoyFilter_HTTP_FILTER
}

func applyToVirtualHost(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.ApplyTo = istionetworkingapi.EnvoyFilter_VIRTUAL_HOST
}

func applyToHttpRoute(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.ApplyTo = istionetworkingapi.EnvoyFilter_HTTP_ROUTE
}

func matchGateway(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.Match = &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectMatch{}
	patch.Match.Context = istionetworkingapi.EnvoyFilter_GATEWAY
}

func matchEnvoyRouter(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	matchGateway(patch)
	patch.Match.ObjectTypes = &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener{}
	objectTypes := patch.Match.ObjectTypes.(*istionetworkingapi.EnvoyFilter_EnvoyConfigObjectMatch_Listener)
	objectTypes.Listener = &istionetworkingapi.EnvoyFilter_ListenerMatch{}
	objectTypes.Listener.FilterChain = &istionetworkingapi.EnvoyFilter_ListenerMatch_FilterChainMatch{}
	objectTypes.Listener.FilterChain.Filter = &istionetworkingapi.EnvoyFilter_ListenerMatch_FilterMatch{}
	objectTypes.Listener.FilterChain.Filter.Name = "envoy.http_connection_manager"
	objectTypes.Listener.FilterChain.Filter.SubFilter = &istionetworkingapi.EnvoyFilter_ListenerMatch_SubFilterMatch{}
	objectTypes.Listener.FilterChain.Filter.SubFilter.Name = "envoy.router"
}

func matchVirtualHost(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch, vhost string) {
	matchGateway(patch)
	patch.Match.ObjectTypes = &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration{}
	objectTypes := patch.Match.ObjectTypes.(*istionetworkingapi.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration)
	objectTypes.RouteConfiguration = &istionetworkingapi.EnvoyFilter_RouteConfigurationMatch{}
	objectTypes.RouteConfiguration.Vhost = &istionetworkingapi.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{}
	objectTypes.RouteConfiguration.Vhost.Name = vhost
}

func matchVirtualHostRoute(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch, vhost, route string) {
	matchVirtualHost(patch, vhost)
	objectTypes := patch.Match.ObjectTypes.(*istionetworkingapi.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration)
	objectTypes.RouteConfiguration.Vhost.Route = &istionetworkingapi.EnvoyFilter_RouteConfigurationMatch_RouteMatch{}
	objectTypes.RouteConfiguration.Vhost.Route.Name = route
}

func insertBefore(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.Patch = &istionetworkingapi.EnvoyFilter_Patch{}
	patch.Patch.Operation = istionetworkingapi.EnvoyFilter_Patch_INSERT_BEFORE
}

func merge(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.Patch = &istionetworkingapi.EnvoyFilter_Patch{}
	patch.Patch.Operation = istionetworkingapi.EnvoyFilter_Patch_MERGE
}
