package config

import (
	"log"
	"strings"
	"time"
)

func (c *config) normalize() {
	c.Controller.normalize()
	c.Service.normalize()
	c.ExtAuthz.normalize()
	c.Keycloak.normalize()
	c.Telemetry.normalize()
}

func (c *controller) normalize() {
	if c.IstioRootNamespace == "" {
		c.IstioRootNamespace = "istio-system"
	}

	if c.EnvoyFilterNamePrefix == "" {
		c.EnvoyFilterNamePrefix = "ext-authz-"
	} else if !strings.HasSuffix(c.EnvoyFilterNamePrefix, "-") {
		c.EnvoyFilterNamePrefix += "-"
	}

	if c.EnvoyFilterLabels == nil {
		c.EnvoyFilterLabels = map[string]string{
			"istio-keycloak": "ext-authz",
		}
	}
}

func (s *service) normalize() {
	if s.Address == "" {
		s.Address = ":8080"
	}
}

func (ea *extAuthz) normalize() {
	if ea.ClusterName == "" {
		log.Fatal("Missing cluster name")
	}

	if ea.Timeout == 0 {
		ea.Timeout = time.Second
	}

	ea.Sessions.normalize()
}

func (s *sessions) normalize() {
	if s.CleaningInterval == 0 {
		s.CleaningInterval = time.Minute
	}

	if s.CleaningGracePeriod == 0 {
		s.CleaningGracePeriod = time.Minute
	}
}

func (k *keycloak) normalize() {
	if k.Url == "" {
		log.Fatal("Missing Keycloak URL")
	}
}

func (t *telemetry) normalize() {
	if t.Address == "" {
		t.Address = ":8081"
	}
}
