package controller

import (
	"github.com/apex/log"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type GatewayMapper struct{}

func (m *GatewayMapper) Map(mapObj handler.MapObject) []reconcile.Request {
	log.WithFields(log.Fields{
		"namespace": mapObj.Meta.GetNamespace(),
		"name": mapObj.Meta.GetName(),
	}).Info("Received update for Gateway")
	return []reconcile.Request{}
}
