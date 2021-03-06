package config

import (
	"time"
)

type config struct {
	Controller  controller  `yaml:"Controller"`
	Service     service     `yaml:"Service"`
	ExtAuthz    extAuthz    `yaml:"ExtAuthz"`
	Sessions    sessions    `yaml:"Sessions"`
	Replication replication `yaml:"Replication"`
	Telemetry   telemetry   `yaml:"Telemetry"`
}

type controller struct {
	IstioRootNamespace string `yaml:"IstioRootNamespace"`

	EnvoyFilterNamePrefix string            `yaml:"EnvoyFilterNamePrefix"`
	EnvoyFilterLabels     map[string]string `yaml:"EnvoyFilterLabels"`

	LeaderElection          bool   `yaml:"LeaderElection"`
	LeaderElectionNamespace string `yaml:"LeaderElectionNamespace"`
	LeaderElectionName      string `yaml:"LeaderElectionName"`
}

type service struct {
	Address string `yaml:"Address"`
}

type extAuthz struct {
	ClusterName string        `yaml:"ClusterName"`
	Timeout     time.Duration `yaml:"Timeout"`
}

type sessions struct {
	CleaningInterval    time.Duration `yaml:"CleaningInterval"`
	CleaningGracePeriod time.Duration `yaml:"CleaningGracePeriod"`
}

const (
	NoneMode   = "none"
	StaticMode = "static"
	DnsMode    = "dns"
)

type replication struct {
	Mode        string                 `yaml:"Mode"`
	StaticPeers []string               `yaml:"StaticPeers"`
	PeerAddress replicationPeerAddress `yaml:"PeerAddress"`

	AdvertiseAddress  string        `yaml:"AdvertiseAddress"`
	EstablishInterval time.Duration `yaml:"EstablishInterval"`
}

type replicationPeerAddress struct {
	Domain  string `yaml:"Domain"`
	Service string `yaml:"Service"`
}

type telemetry struct {
	Address string `yaml:"Address"`
}
