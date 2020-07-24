package controller

import (
	"github.com/KnowitSolutions/istio-oidc/controller/accesspolicy"
	"github.com/KnowitSolutions/istio-oidc/controller/envoyfilter"
	"github.com/KnowitSolutions/istio-oidc/controller/secret"
	"github.com/KnowitSolutions/istio-oidc/state"
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
