package config

import (
	"github.com/apex/log"
	"gopkg.in/yaml.v3"
	"os"
)

func Load(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.WithError(err).WithField("filename", filename).
			Fatal("Unable to load configuration")
	}
	defer func() { _ = file.Close() }()

	cfg := config{}
	dec := yaml.NewDecoder(file)
	err = dec.Decode(&cfg)
	if err != nil {
		log.WithError(err).WithField("filename", filename).
			Fatal("Unable to load configuration")
	}

	cfg.normalize()

	Controller = cfg.Controller
	Service = cfg.Service
	ExtAuthz = cfg.ExtAuthz
	Sessions = cfg.ExtAuthz.Sessions
	Keycloak = cfg.Keycloak
	Telemetry = cfg.Telemetry
}
