package sync

import (
	"context"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"net"
)

type stream interface {
	Send(*Message) error
	Recv() (*Message, error)
	Context() context.Context
}

type peer struct {
	peers  *peerSet
	ip     net.IP
	stream stream
	send   chan<- *Message
	recv   <-chan *Message
	done   chan struct{}
	close  func()
}

func newPeer(peers *peerSet, ip net.IP, stream stream) *peer {
	peer := peer{peers: peers, ip: ip, stream: stream}
	peer.done = make(chan struct{})
	go send(&peer)
	go recv(&peer)
	return &peer
}

func send(peer *peer) {
	ch := make(chan *Message)
	peer.send = ch

loop:
	for {
		select {
		case msg := <-ch:
			err := peer.stream.Send(msg)
			if err != nil {
				peer.disconnect(err)
			}
		case <-peer.done:
			break loop
		}
	}
}

func recv(peer *peer) {
	in := make(chan *Message)
	out := make(chan *Message)
	peer.recv = out
	go recvFwd(peer, in)

loop:
	for {
		select {
		case msg := <-in:
			out <- msg
		case <-peer.done:
			break loop
		}
	}

	close(out)
}

func recvFwd(peer *peer, ch chan<- *Message) {
	for {
		msg, err := peer.stream.Recv()
		if err != nil {
			peer.disconnect(err)
			break
		} else {
			ch <- msg
		}
	}
}

func (p *peer) disconnect(err error) {
	if p.stream == nil {
		return
	} else if err != nil {
		err = errors.Wrap(err, "", "peer", p.ip.String())
		log.Error(nil, err, "Lost connection")
	} else {
		vals := log.MakeValues("peer", p.ip.String())
		log.Info(nil, vals, "Disconnecting")
	}

	p.peers.remove(ipToKey(p.ip))
	close(p.done)
	p.stream = nil
	if p.close != nil {
		p.close()
	}
}
