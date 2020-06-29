package controller

import (
	"istio-keycloak/controller/accesspolicy"
	"istio-keycloak/controller/envoyfilter"
	"istio-keycloak/state"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Register(mgr ctrl.Manager, oidcCommStore state.OidcCommunicatorStore) error {
	err := accesspolicy.New(mgr, oidcCommStore)
	if err != nil {
		return err
	}

	err = envoyfilter.New(mgr)
	if err != nil {
		return err
	}

	return nil
}
