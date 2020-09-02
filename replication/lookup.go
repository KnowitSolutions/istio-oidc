package replication

import (
	"context"
	"fmt"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"net"
)

type endpointLookup interface {
	lookupEndpoints(context.Context) ([]string, error)
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
	return config.Replication.StaticPeers, nil
}

type dnsEndpoints struct{}

func (dnsEndpoints) lookupEndpoints(ctx context.Context) ([]string, error) {
	svc := config.Replication.PeerAddress.Service
	name := config.Replication.PeerAddress.Domain

	_, addrs, err := net.DefaultResolver.LookupSRV(ctx, svc, "tcp", name)
	if err != nil {
		err := errors.Wrap(err, "failed looking up endpoints")
		return nil, err
	}

	eps := make([]string, len(addrs))
	for i := range addrs {
		eps[i] = fmt.Sprintf("%s:%d", addrs[i].Target, addrs[i].Port)
	}
	return eps, nil
}

type noneEndpoints struct{}

func (noneEndpoints) lookupEndpoints(_ context.Context) ([]string, error) {
	return nil, nil
}
