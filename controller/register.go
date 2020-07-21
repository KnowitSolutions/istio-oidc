package controller

import (
	"istio-keycloak/controller/accesspolicy"
	"istio-keycloak/controller/envoyfilter"
	"istio-keycloak/controller/secret"
	"istio-keycloak/state"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Register(mgr ctrl.Manager, keyStore state.KeyStore, apStore state.AccessPolicyStore) error {
	err := accesspolicy.Register(mgr, apStore)
	if err != nil {
		return err
	}

	err = envoyfilter.Register(mgr)
	if err != nil {
		return err
	}

	err = secret.Register(mgr, keyStore)
	if err != nil {
		return err
	}

	return nil
}
