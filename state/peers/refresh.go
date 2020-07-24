package peers

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"google.golang.org/grpc"
	"net"
	"strings"
	"time"
)

func background(peers *peerSet) {
	ch := time.Tick(10 * time.Second) // TODO: Configurable
	for {
		<-ch
		refresh(peers)
	}
}

func refresh(peers *peerSet) {
	ctx := context.Background()

	ownIp, peerIps, err := getIps(ctx)
	allow := make(map[peerKey]bool, len(peerIps))
	if err != nil {
		log.Error(ctx, err, "Failed refreshing peers")
		return
	}

	for _, peerIp := range peerIps {
		key := ipToKey(peerIp)
		peers.allow(key)
		allow[key] = true

		isConnected := peers.get(key) != nil
		shouldConnect := strings.Compare(string(ownIp), string(peerIp)) < 0
		if isConnected || !shouldConnect {
			continue
		}

		err = connect(peers, peerIp)
		if err != nil {
			log.Error(ctx, err, "Failed connecting to peer")
		}
	}

	peerKeys := peers.list()
	for _, key := range peerKeys {
		if allow[key] {
			continue
		}

		peer := peers.get(key)
		peer.disconnect(nil)
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
		own := addr.(*net.IPNet)
		if own != nil && ip.IP.Equal(own.IP) {
			return true
		}
	}
	return false
}

func connect(peers *peerSet, ip net.IP) error {
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
	peer.close = func() {
		_ = stream.CloseSend()
		_ = conn.Close()
	}

	vals := log.MakeValues("peer", ip.String())
	log.Info(ctx, vals, "Connected to peer")

	go peers.talk(peer)
	return nil
}
