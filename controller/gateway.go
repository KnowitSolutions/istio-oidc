package controller

import (
	"fmt"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func virtualHosts(gateway *istionetworking.Gateway) []string {
	count := 0
	for _, srv := range gateway.Spec.Servers {
		count += len(srv.Hosts)
	}

	vhosts := make([]string, 0, count)
	for _, srv := range gateway.Spec.Servers {
		for _, host := range srv.Hosts {
			vhost := fmt.Sprintf("%s:%d", host, srv.Port.Number)
			vhosts = append(vhosts, vhost)
		}
	}

	return vhosts
}
