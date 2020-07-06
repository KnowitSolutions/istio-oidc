package envoyfilter

import (
	"fmt"
	"istio-keycloak/config"
	"istio-keycloak/state/accesspolicy"
	istionetworkingapi "istio.io/api/networking/v1alpha3"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func newEnvoyFilter(ef *istionetworking.EnvoyFilter, pols []*accesspolicy.AccessPolicy) {
	count := 2
	for _, pol := range pols {
		count += len(pol.VirtualHosts) * len(pol.Routes)
	}
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
	extAuthzPerRoute(extAuthzDisable, "", "", nil)
	ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, extAuthzDisable)

	for _, pol := range pols {
		for _, vhost := range pol.VirtualHosts {
			patch := &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch{}
			applyToVirtualHost(patch)
			matchVirtualHost(patch, vhost)
			merge(patch)
			extAuthzPerRoute(patch, pol.Name, "", &pol.Default)

			ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, patch)

			for route, routeData := range pol.Routes {
				patch := &istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch{}
				applyToHttpRoute(patch)
				matchVirtualHostRoute(patch, vhost, route)
				merge(patch)
				extAuthzPerRoute(patch, pol.Name, route, &routeData)

				ef.Spec.ConfigPatches = append(ef.Spec.ConfigPatches, patch)
			}
		}
	}
}

func extAuthz(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch) {
	patch.Patch.Value = newStruct(map[string]interface{}{
		"name": "envoy.filters.http.ext_authz",
		"typed_config": map[string]interface{}{
			"@type": "type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthz",
			"grpc_service": map[string]interface{}{
				"envoy_grpc": map[string]interface{}{
					"cluster_name": config.ExtAuthz.ClusterName,
				},
				"timeout": fmt.Sprintf("%.fs", config.ExtAuthz.Timeout.Seconds()),
			},
		},
	})
}

func extAuthzPerRoute(patch *istionetworkingapi.EnvoyFilter_EnvoyConfigObjectPatch, policy, route string, routeData *accesspolicy.Route) {
	cfg := map[string]interface{}{
		"@type": "type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthzPerRoute",
	}

	if routeData != nil && routeData.EnableAuthz {
		cfg["check_settings"] = map[string]interface{}{
			"context_extensions": map[string]interface{}{
				accesspolicy.NameKey:  policy,
				accesspolicy.RouteKey: route,
			},
		}
	} else {
		cfg["disabled"] = true
	}

	patch.Patch.Value = newStruct(map[string]interface{}{
		"typed_per_filter_config": map[string]interface{}{
			"envoy.filters.http.ext_authz": cfg,
		},
	})
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
