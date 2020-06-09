package controller

import (
	"istio-keycloak/config"
)

type service struct {
	*config.Service
	ingress ingress
	vhosts  []string
}