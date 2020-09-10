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
	eps         map[string]string
	conns       map[string]*connection
	connsMu     sync.RWMutex
	handshakeMu sync.Mutex
	lookup      endpointLookup
}

func NewPeers() *Peers {
	return &Peers{
		eps:    map[string]string{},
		conns:  map[string]*connection{},
		lookup: newEndpointLookup(),
	}
}

func (p *Peers) hasEp(ep string) bool {
	p.connsMu.RLock()
	defer p.connsMu.RUnlock()

	_, ok := p.eps[ep]
	return ok
}

func (p *Peers) hasPeer(id string) bool {
	p.connsMu.RLock()
	defer p.connsMu.RUnlock()

	_, ok := p.conns[id]
	return ok
}

func (p *Peers) getEps() []string {
	p.connsMu.RLock()
	defer p.connsMu.RUnlock()

	eps := make([]string, 0, len(p.eps))
	for ep := range p.eps {
		eps = append(eps, ep)
	}
	return eps
}

func (p *Peers) getPeers() []*connection {
	p.connsMu.RLock()
	defer p.connsMu.RUnlock()

	conns := make([]*connection, 0, len(p.conns))
	for _, conn := range p.conns {
		conns = append(conns, conn)
	}
	return conns
}

func (p *Peers) addUnsafe(conn *connection) {
	p.eps[conn.id] = conn.ep
	p.conns[conn.id] = conn
}

func (p *Peers) removeUnsafe(ep string) {
	id := p.eps[ep]
	conn := p.conns[id]
	if conn != nil {
		_ = conn.conn.Close()
	}

	delete(p.eps, ep)
	delete(p.conns, id)
}

func (p *Peers) refresh(ctx context.Context) error {
	eps, err := p.lookup.lookupEndpoints(ctx)
	if err != nil {
		err := errors.Wrap(err, "failed refreshing endpoints")
		return err
	}

	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	keep := make(map[string]bool, len(eps))
	for _, ep := range eps {
		keep[ep] = true
		p.eps[ep] = p.eps[ep]
	}

	for ep := range p.eps {
		if !keep[ep] {
			p.removeUnsafe(ep)
		}
	}

	return nil
}

func (p *Peers) getConnection(ctx context.Context, self *Self, peer string) *connection {
	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	id := p.eps[peer]
	conn := p.conns[id]
	if conn == nil {
		conn = newConnection(ctx, self, peer, &p.handshakeMu)
		p.addUnsafe(conn)
	}

	return conn
}
