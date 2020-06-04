package controller

import (
	"fmt"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"reflect"
)

var defaultIngressSelector = map[string]string{"istio": "ingressgateway"}

func mkIngress(gateway *istionetworking.Gateway) ingress {
	i := ingress{selector: gateway.Spec.Selector}
	if reflect.DeepEqual(i.selector, defaultIngressSelector) {
		i.namespace = IstioRootNamespace
	} else {
		i.namespace = gateway.Namespace
	}
	return i
}

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
