package controller

import (
	"github.com/KnowitSolutions/istio-oidc/controller/accesspolicy"
	"github.com/KnowitSolutions/istio-oidc/controller/envoyfilter"
	apstate "github.com/KnowitSolutions/istio-oidc/state/accesspolicy"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Register(mgr ctrl.Manager, apStore apstate.Store) error {
	err := accesspolicy.Register(mgr, apStore)
	if err != nil {
		return err
	}

	err = envoyfilter.Register(mgr)
	if err != nil {
		return err
	}

	return nil
}
