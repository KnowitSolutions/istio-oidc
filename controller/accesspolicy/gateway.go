package accesspolicy

import (
	"fmt"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func selector(gw *istionetworking.Gateway) map[string]string {
	selector := make(map[string]string, len(gw.Spec.Selector))
	for k, v := range gw.Spec.Selector {
		selector[k] = v
	}
	return selector
}

func virtualHosts(gw *istionetworking.Gateway) []string {
	count := 0
	for _, srv := range gw.Spec.Servers {
		count += len(srv.Hosts)
	}

	vhosts := make([]string, 0, count)
	for _, srv := range gw.Spec.Servers {
		for _, host := range srv.Hosts {
			vhost := fmt.Sprintf("%s:%d", host, srv.Port.Number)
			vhosts = append(vhosts, vhost)
		}
	}

	return vhosts
}
