package controller

import (
	"istio-keycloak/api/v1"
	"istio-keycloak/config"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

type accessPolicy struct {
	*config.AccessPolicy
	ingress ingress
	vhosts  []string
}

func newAccesPolicy(ap *api.AccessPolicy, cred *core.Secret, gw *istionetworking.Gateway) accessPolicy {
	return accessPolicy{
		AccessPolicy: config.NewAccessPolicy(ap, cred),
		ingress:      *newIngress(gw),
		vhosts:       virtualHosts(gw),
	}
}

func credentialsKey(ap *api.AccessPolicy) types.NamespacedName {
	return types.NamespacedName{Namespace: ap.Namespace, Name: ap.Spec.OIDC.CredentialsSecret.Name}
}

func gatewayKey(ap *api.AccessPolicy) types.NamespacedName {
	parts := strings.SplitN(ap.Spec.Gateway, "/", 2)
	if len(parts) == 2 {
		return types.NamespacedName{Namespace: parts[0], Name: parts[1]}
	} else {
		return types.NamespacedName{Namespace: ap.Namespace, Name: parts[0]}
	}
}
