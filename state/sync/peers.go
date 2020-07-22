package sync

import (
	"context"
	"google.golang.org/grpc"
	grpcpeer "google.golang.org/grpc/peer"
	"istio-keycloak/config"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"net"
	"strings"
	"sync"
	"time"
)

type stream interface {
	Send(*Message) error
	Recv() (*Message, error)
	Context() context.Context
}

type openFunc func(context.Context, *grpc.ClientConn) (stream, error)
type talkFunc func(peer)
type peers struct {
	all  map[[net.IPv6len]byte]peer
	mu   sync.Mutex
	open openFunc
	talk talkFunc
}

func ipToKey(ip net.IP) [net.IPv6len]byte {
	var key [net.IPv6len]byte
	copy(key[:], ip.To16())
	return key
}

func newPeers(open openFunc, talk talkFunc) *peers {
	peers := peers{
		all:  map[[net.IPv6len]byte]peer{},
		open: open,
		talk: talk,
	}
	go background(&peers)
	return &peers
}

func (p *peers) add(stream stream) (peer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	meta, ok := grpcpeer.FromContext(stream.Context())
	if !ok {
		return peer{}, errors.New("missing gRPC peer data")
	}

	peer := peer{
		active: true,
		ip:     meta.Addr.(*net.TCPAddr).IP,
		peers:  p,
		stream: stream,
	}

	key := ipToKey(peer.ip)
	_, ok = p.all[key]
	if ok {
		p.all[key] = peer
		return peer, nil
	} else {
		return peer, errors.New("peer IP not recognized")
	}
}

func (p *peers) send(msg *Message) {
	for _, peer := range p.all {
		go peer.send(msg)
	}
}

func background(peers *peers) {
	ch := time.Tick(10 * time.Second) // TODO: Configurable
	for {
		<-ch
		peers.refresh()
	}
}

func (p *peers) refresh() {
	ctx := context.Background()
	ownIp, peerIps, err := getIps(ctx)
	if err != nil {
		log.Error(ctx, err, "Failed refreshing peers")
		return
	}

	ownStr := string(ownIp)
	for _, peerIp := range peerIps {
		p.mu.Lock()
		key := ipToKey(peerIp)
		if p.all[key].active {
			continue
		} else {
			p.all[key] = peer{active: false}
		}
		p.mu.Unlock()

		peerStr := string(peerIp)
		if strings.Compare(ownStr, peerStr) > 0 {
			continue
		}

		err = connect(p, peerIp)
		if err != nil {
			log.Error(ctx, err, "Failed connecting to peer")
		}
	}
}

func getIps(ctx context.Context) (net.IP, []net.IP, error) {
	ownAddrs, err := net.InterfaceAddrs()
	if err != nil {
		err = errors.Wrap(err, "failed getting own IPs")
		return net.IP{}, nil, err
	}

	dnsIps, err := net.DefaultResolver.LookupIPAddr(ctx, config.Service.Hostname)
	if err != nil {
		err = errors.Wrap(err, "failed resolving peer IPs", "address", "")
		return net.IP{}, nil, err
	}

	ownIps := make([]net.IP, 0, len(dnsIps))
	peerIps := make([]net.IP, 0, len(dnsIps))
	for _, ip := range dnsIps {
		if isOwnIp(ip, ownAddrs) {
			ownIps = append(ownIps, ip.IP.To16())
		} else {
			peerIps = append(peerIps, ip.IP.To16())
		}
	}

	if len(ownIps) != 1 {
		err = errors.New("exactly one IP from DNS needs to be owned by this machine")
		return net.IP{}, nil, err
	}

	return ownIps[0], peerIps, nil
}

func isOwnIp(ip net.IPAddr, own []net.Addr) bool {
	for _, addr := range own {
		own := addr.(*net.IPAddr)
		if own != nil && ip.IP.Equal(own.IP) {
			return true
		}
	}
	return false
}

func connect(peers *peers, ip net.IP) error {
	ctx := context.Background()

	idx := strings.LastIndexByte(config.Service.Address, ':')
	addr := ip.String() + config.Service.Address[idx:]
	conn, err := grpc.DialContext(ctx, addr) // TODO: Maybe enable keepalive?
	if err != nil {
		err = errors.Wrap(err, "", "peer", addr)
		return err
	}

	stream, err := peers.open(ctx, conn)
	if err != nil {
		err = errors.Wrap(err, "", "peer", addr)
		return err
	}

	peer, err := peers.add(stream)
	if err != nil {
		return errors.Wrap(err, "", "peer", addr)
	}

	go func() { peers.talk(peer); _ = conn.Close() }()
	return nil
}

type peer struct {
	active bool
	ip     net.IP
	peers  *peers
	stream stream
	sendMu *sync.Mutex
	recvMu *sync.Mutex
}

func (p *peer) send(msg *Message) bool {
	p.sendMu.Lock()
	defer p.sendMu.Unlock()

	err := p.stream.Send(msg)
	if err != nil {
		err = errors.Wrap(err, "", "peer", p.ip.String())
		log.Error(nil, err, "Lost connection")
		remove(p)
		return false
	} else {
		return true
	}
}

func (p *peer) recv() *Message {
	p.recvMu.Lock()
	defer p.recvMu.Unlock()

	msg, err := p.stream.Recv()
	if err != nil {
		err = errors.Wrap(err, "", "peer", p.ip.String())
		log.Error(nil, err, "Lost connection")
		remove(p)
		return nil
	} else {
		return msg
	}
}

func remove(peer *peer) {
	peer.peers.mu.Lock()
	defer peer.peers.mu.Unlock()
	delete(peer.peers.all, ipToKey(peer.ip))
}
