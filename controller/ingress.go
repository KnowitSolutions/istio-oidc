package controller

import (
	"bytes"
	"encoding/binary"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"reflect"
	"strings"
)

var defaultIngressSelector = map[string]string{"istio": "ingressgateway"}

type ingress struct {
	key       string
	namespace string
	selector  map[string]string
}

func newIngress(gateway *istionetworking.Gateway) *ingress {
	var b bytes.Buffer
	for k, v := range gateway.Spec.Selector {
		_ = binary.Write(&b, binary.LittleEndian, len(k))
		_ = binary.Write(&b, binary.LittleEndian, k)
		_ = binary.Write(&b, binary.LittleEndian, len(v))
		_ = binary.Write(&b, binary.LittleEndian, v)
	}

	var ns string
	if reflect.DeepEqual(gateway.Spec.Selector, defaultIngressSelector) {
		ns = IstioRootNamespace
	} else {
		ns = gateway.Namespace
	}

	return &ingress{
		key:       b.String(),
		namespace: ns,
		selector:  gateway.Spec.Selector,
	}
}

func (i *ingress) String() string {
	var b strings.Builder
	b.WriteString(i.namespace)
	b.WriteRune('/')
	b.WriteString(EnvoyFilterNamePrefix)
	b.WriteRune('*')
	b.WriteRune('{')

	first := true
	for k, v := range i.selector {
		if first {
			first = false
		} else {
			b.WriteRune(',')
		}

		b.WriteString(k)
		b.WriteRune('=')
		b.WriteString(v)
	}

	b.WriteRune('}')
	return b.String()
}

func ingresses(pols []accessPolicy) []*ingress {
	hash := make(map[string]*ingress, len(pols))
	for _, pol := range pols {
		hash[pol.ingress.key] = &pol.ingress
	}

	i := 0
	list := make([]*ingress, len(hash))
	for _, ingress := range hash {
		list[i] = ingress
		i++
	}

	return list
}
