package config

import (
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func (c *config) normalize() {
	c.Controller.normalize()
	c.Service.normalize()
	c.ExtAuthz.normalize()
	c.Sessions.normalize()
	c.Replication.normalize()
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
			"istio-oidc": "ext-authz",
		}
	}

	if c.LeaderElectionNamespace == "" {
		ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			err = errors.New("missing leader election namespace")
			log.Error(nil, err, "Failed loading config")
			os.Exit(1)
		}

		c.LeaderElectionNamespace = string(ns)
	}

	if c.LeaderElectionName == "" {
		c.LeaderElectionName = "istio-oidc"
	}

	if c.TokenKeyNamespace == "" {
		ns, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			err = errors.New("missing token key namespace")
			log.Error(nil, err, "Failed loading config")
			os.Exit(1)
		}

		c.TokenKeyNamespace = string(ns)
	}

	if c.TokenKeyName == "" {
		c.TokenKeyName = "istio-oidc"
	}
}

func (s *service) normalize() {
	if s.Address == "" {
		s.Address = ":8080"
	}
}

func (ea *extAuthz) normalize() {
	if ea.ClusterName == "" {
		err := errors.New("missing cluster name")
		log.Error(nil, err, "Failed loading config")
		os.Exit(1)
	}

	if ea.Timeout == 0 {
		ea.Timeout = time.Second
	}
}

func (s *sessions) normalize() {
	if s.CleaningInterval == 0 {
		s.CleaningInterval = time.Minute
	}

	if s.CleaningGracePeriod == 0 {
		s.CleaningGracePeriod = time.Minute
	}
}
func (r *replication) normalize() {
	switch r.Mode {
	case "":
		r.Mode = NoneMode
	case NoneMode:
	case StaticMode:
	case DnsMode:
	default:
		err := errors.New("invalid replication mode")
		log.Error(nil, err, "Failed loading config")
		os.Exit(1)
	}

	// TODO: Load from pod IP interface
	if r.AdvertiseAddress == "" {
		err := errors.New("missing advertise address")
		log.Error(nil, err, "Failed loading config")
		os.Exit(1)
	}

	if r.EstablishInterval == 0 {
		r.EstablishInterval = time.Minute
	}
}

func (k *keycloak) normalize() {
	if k.Url == "" {
		err := errors.New("missing Keycloak URL")
		log.Error(nil, err, "Failed loading config")
		os.Exit(1)
	}
}

func (t *telemetry) normalize() {
	if t.Address == "" {
		t.Address = ":8081"
	}
}
