package peers

import (
	"context"
	"google.golang.org/grpc"
	grpcpeer "google.golang.org/grpc/peer"
	"istio-keycloak/log/errors"
	"net"
	"sync"
)

type peerKey [net.IPv6len]byte
type openFunc func(context.Context, *grpc.ClientConn) (Peering_StreamClient, error)
type talkFunc func(*peer)
type peerSet struct {
	set  map[peerKey]*peer
	mu   sync.RWMutex
	open openFunc
	talk talkFunc
}

func ipToKey(ip net.IP) peerKey {
	var key peerKey
	copy(key[:], ip.To16())
	return key
}

func newPeerSet(open openFunc, talk talkFunc) *peerSet {
	set := map[peerKey]*peer{}
	peers := peerSet{set: set, open: open, talk: talk}
	go background(&peers)
	return &peers
}

func (p *peerSet) allow(key peerKey) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.set[key]; !ok {
		p.set[key] = nil
	}
}

func (p *peerSet) add(stream stream) (*peer, error) {
	meta, ok := grpcpeer.FromContext(stream.Context())
	if !ok {
		return nil, errors.New("missing gRPC peer data")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	ip := meta.Addr.(*net.TCPAddr).IP
	key := ipToKey(ip)
	_, ok = p.set[key]
	if !ok {
		return nil, errors.New("peer IP not recognized")
	}

	p.set[key] = newPeer(p, ip, stream)
	return p.set[key], nil
}

func (p *peerSet) list() []peerKey {
	p.mu.RLock()
	defer p.mu.RUnlock()

	list := make([]peerKey, 0, len(p.set))
	for k := range p.set {
		list = append(list, k)
	}
	return list
}

func (p *peerSet) get(key peerKey) *peer {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.set[key]
}

func (p *peerSet) remove(key peerKey) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.set, key)
}

func (p *peerSet) send(msg *Message) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	send := func(peer *peer) { peer.send <- msg }
	for _, peer := range p.set {
		if peer != nil {
			go send(peer)
		}
	}
}
