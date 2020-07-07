package config

import (
	"gopkg.in/yaml.v3"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"os"
)

func Load(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		err = errors.Wrap(err, "", "filename", filename)
		log.Error(nil, err, "Unable to load configuration")
		os.Exit(1)
	}
	defer func() { _ = file.Close() }()

	cfg := config{}
	dec := yaml.NewDecoder(file)
	err = dec.Decode(&cfg)
	if err != nil {
		err = errors.Wrap(err, "", "filename", filename)
		log.Error(nil, err, "Unable to load configuration")
		os.Exit(1)
	}

	cfg.normalize()

	Controller = cfg.Controller
	Service = cfg.Service
	ExtAuthz = cfg.ExtAuthz
	Sessions = cfg.ExtAuthz.Sessions
	Keycloak = cfg.Keycloak
	Telemetry = cfg.Telemetry
}
