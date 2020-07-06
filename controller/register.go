package controller

import (
	"istio-keycloak/controller/accesspolicy"
	"istio-keycloak/controller/envoyfilter"
	"istio-keycloak/state"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Register(mgr ctrl.Manager, apStore state.AccessPolicyStore) error {
	err := accesspolicy.New(mgr, apStore)
	if err != nil {
		return err
	}

	err = envoyfilter.New(mgr)
	if err != nil {
		return err
	}

	return nil
}
