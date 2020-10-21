package replication

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

func (p *Peers) refresh(ctx context.Context) ([]string, error) {
	eps, err := p.lookup.lookupEndpoints(ctx)
	if err != nil {
		err := errors.Wrap(err, "failed refreshing endpoints")
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	keep := make(map[string]bool, len(eps))
	for _, ep := range eps {
		keep[ep] = true
	}

	for ep, conn := range p.conns {
		if !keep[ep] {
			conn.disconnect()
			delete(p.conns, ep)
		}
	}

	return eps, nil
}

func (p *Peers) getConnection(self *Self, ep string) (*connection, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conns[ep] == nil {
		authority := p.lookup.authority(ep)
		p.conns[ep] = newConnection(self, ep, authority)

		return p.conns[ep], true
	} else {
		return p.conns[ep], false
	}
}

func (p *Peers) getConnections() []*connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conns := make([]*connection, 0, len(p.conns))
	for _, conn := range p.conns {
		conns = append(conns, conn)
	}
	return conns
}
