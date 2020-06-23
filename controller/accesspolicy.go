package controller

import (
	"fmt"
	"istio-keycloak/api/v1"
	"istio-keycloak/state"
	istionetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

type accessPolicy struct {
	*state.AccessPolicy
	key         string
	credentials types.UID
	gateway     types.UID
	ingress     ingress
	vhosts      []string
}

func newAccessPolicy(ap *api.AccessPolicy, cred *core.Secret, gw *istionetworking.Gateway) (*accessPolicy, error) {
	apCfg, err := state.NewAccessPolicy(ap, cred)
	if err != nil {
		return nil, err
	}

	return &accessPolicy{
		AccessPolicy: apCfg,
		key:          fmt.Sprintf("%s/%s", ap.Namespace, ap.Name),
		credentials:  cred.GetUID(),
		gateway:      gw.GetUID(),
		ingress:      *newIngress(gw),
		vhosts:       virtualHosts(gw),
	}, nil
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
