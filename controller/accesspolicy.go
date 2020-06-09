package controller

import (
	"istio-keycloak/config"
)

type accessPolicy struct {
	*config.AccessPolicy
	ingress ingress
	vhosts  []string
}