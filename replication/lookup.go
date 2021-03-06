package replication

import (
	"context"
	"fmt"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"net"
)

type endpointLookup interface {
	lookupEndpoints(context.Context) ([]string, error)
	authority(string) string
}

func newEndpointLookup() endpointLookup {
	switch config.Replication.Mode {
	case config.StaticMode:
		return staticEndpoints{}
	case config.DnsMode:
		return dnsEndpoints{}
	case config.NoneMode:
		return noneEndpoints{}
	default:
		panic("invalid endpoints mode")
	}
}

type staticEndpoints struct{}

func (staticEndpoints) lookupEndpoints(_ context.Context) ([]string, error) {
	self := config.Replication.AdvertiseAddress
	peers := config.Replication.StaticPeers
	eps := make([]string, 0, len(peers))

	for _, ep := range peers {
		if ep != self {
			eps = append(eps, ep)
		}
	}

	return eps, nil
}

func (staticEndpoints) authority(addr string) string {
	return addr
}

type dnsEndpoints struct{}

func (dnsEndpoints) lookupEndpoints(ctx context.Context) ([]string, error) {
	self := config.Replication.AdvertiseAddress
	svc := config.Replication.PeerAddress.Service
	name := config.Replication.PeerAddress.Domain

	_, srvs, err := net.DefaultResolver.LookupSRV(ctx, svc, "tcp", name)
	dnsErr, _ := err.(*net.DNSError)
	if dnsErr != nil && dnsErr.IsNotFound {
		log.Error(ctx, dnsErr, "No peer information available")
		return nil, nil
	} else if err != nil {
		err := errors.Wrap(err, "failed looking up endpoints")
		return nil, err
	}

	eps := make([]string, 0, len(srvs))

	for _, srv := range srvs {
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, srv.Target)
		if err != nil {
			err := errors.Wrap(err, "failed resolving endpoint")
			return nil, err
		}

		for _, ip := range ips {
			ep := fmt.Sprintf("%s:%d", ip.IP, srv.Port)
			if ep != self {
				eps = append(eps, ep)
			}
		}
	}

	return eps, nil
}

func (dnsEndpoints) authority(_ string) string {
	return config.Replication.PeerAddress.Domain
}

type noneEndpoints struct{}

func (noneEndpoints) lookupEndpoints(_ context.Context) ([]string, error) {
	return nil, nil
}

func (noneEndpoints) authority(_ string) string {
	return ""
}
