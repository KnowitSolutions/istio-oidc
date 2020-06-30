package config

import (
	"time"
)

type config struct {
	Controller controller `yaml:"Controller"`
	Service    service    `yaml:"Service"`
	ExtAuthz   extAuthz   `yaml:"ExtAuthz"`
	Keycloak   keycloak   `yaml:"Keycloak"`
	Telemetry  telemetry  `yaml:"Telemetry"`
}

type controller struct {
	IstioRootNamespace    string            `yaml:"IstioRootNamespace"`
	EnvoyFilterNamePrefix string            `yaml:"EnvoyFilterNamePrefix"`
	EnvoyFilterLabels     map[string]string `yaml:"EnvoyFilterLabels"`
}

type service struct {
	Address string `yaml:"Address"`
}

type extAuthz struct {
	ClusterName string        `yaml:"ClusterName"`
	Timeout     time.Duration `yaml:"Timeout"`
	Sessions    sessions      `yaml:"Sessions"`
}

type sessions struct {
	CleaningInterval    time.Duration `yaml:"CleaningInterval"`
	CleaningGracePeriod time.Duration `yaml:"CleaningGracePeriod"`
}

type keycloak struct {
	Url string `yaml:"URL"`
}

type telemetry struct {
	Address string `yaml:"Address"`
}
