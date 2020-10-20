package replication

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"sync"
)

func NewPeerId() (string, error) {
	var buf [16]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		err := errors.Wrap(err, "failed generating peer ID")
		return "", err
	}

	id := hex.EncodeToString(buf[:])
	return id, nil
}

type Peers struct {
	conns  map[string]*connection
	mu     sync.RWMutex
	lookup endpointLookup
}

func NewPeers() *Peers {
	return &Peers{
		conns:  map[string]*connection{},
		lookup: newEndpointLookup(),
	}
}

func (p *Peers) getEps() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	eps := make([]string, 0, len(p.conns))
	for ep := range p.conns {
		eps = append(eps, ep)
	}
	return eps
}

func (p *Peers) refresh(ctx context.Context) error {
	eps, err := p.lookup.lookupEndpoints(ctx)
	if err != nil {
		err := errors.Wrap(err, "failed refreshing endpoints")
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	keep := make(map[string]bool, len(eps))
	for _, ep := range eps {
		if ep != config.Replication.AdvertiseAddress {
			keep[ep] = true
			p.conns[ep] = p.conns[ep]
		}
	}
	for ep := range p.conns {
		if !keep[ep] {
			p.conns[ep].disconnect()
			delete(p.conns, ep)
		}
	}

	return nil
}

func (p *Peers) getConnection(self *Self, ep string) *connection {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conns[ep] == nil {
		p.conns[ep] = newConnection(self, ep)
	}
	return p.conns[ep]
}

func (p *Peers) getConnections() []*connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conns := make([]*connection, 0, len(p.conns))
	for _, conn := range p.conns {
		if conn != nil {
			conns = append(conns, conn)
		}
	}
	return conns
}
