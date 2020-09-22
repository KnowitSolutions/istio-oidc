package replication

import (
	"context"
	"encoding/hex"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/state"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"strings"
	"sync"
)

type connection struct {
	ep    string
	conn  *grpc.ClientConn
	peers map[string]struct{}

	live bool
	init chan struct{}
	once sync.Once
}

func newConnection(self *Self, peers map[string]struct{}, peer string) *connection {
	conn := connection{ep: peer, peers: peers, init: make(chan struct{})}
	conn.conn, _ = grpc.Dial(peer, grpc.WithInsecure())

	ctx := log.WithValues(nil, "address", peer)
	go conn.logConnectionState(ctx)
	go conn.handshake(ctx, self)

	return &conn
}

func (c *connection) handshake(ctx context.Context, self *Self) {
	log.Info(ctx, nil, "Handshaking with peer")

	req := api.HandshakeRequest{
		PeerId:       self.id,
		PeerEndpoint: self.ep,
		Latest:       latestToProto(self.copyLatest()),
	}

	client := api.NewReplicationClient(c.conn)
	res, err := client.Handshake(ctx, &req)
	if status.Code(err) == codes.PermissionDenied {
		log.Error(ctx, err, "Permission denied during handshake with peer")
		os.Exit(1)
	} else if err != nil {
		log.Error(ctx, err, "Failed handshaking with peer")
		go c.reestablish(ctx, self, err)
		return
	}

	ctx = log.WithValues(ctx, "peer", res.PeerId)
	c.peers[res.PeerId] = struct{}{}

	go c.update(ctx, self, res.Latest)
}

func (c *connection) reestablish(ctx context.Context, self *Self, err error) {
	var cont bool
	c.once.Do(func() { cont = true })
	if !cont {
		return
	}

	c.live = false
	ch := c.init
	if ch != nil {
		close(ch)
	}

	if status.Code(err) == codes.Canceled {
		log.Info(ctx, nil, "Disconnected from peer")
		return
	}

	log.Info(ctx, nil, "Waiting for transport to recover")
	for c.conn.GetState() == connectivity.TransientFailure {
		c.conn.WaitForStateChange(ctx, connectivity.TransientFailure)
		c.conn.WaitForStateChange(ctx, connectivity.Connecting)
	}

	c.init = make(chan struct{})
	c.once = sync.Once{}

	go c.handshake(ctx, self)
}

func (c *connection) wakeup() {
	c.conn.ResetConnectBackoff()
}

func (c *connection) update(ctx context.Context, self *Self, latest []*api.Stamp) {
	mapped := latestFromProto(latest)
	update := self.needsUpdate(mapped)
	if update {
		log.Info(ctx, nil, "Peer reports new sessions")
		c.streamSessions(ctx, self)
	} else {
		log.Info(ctx, nil, "Peer reports no new sessions")
	}
}

func (c *connection) setSession(ctx context.Context, self *Self, sess state.StampedSession) {
	req := api.SetSessionRequest{
		PeerId:  self.id,
		Session: sessionToProto(sess.Session),
		Stamp:   stampToProto(sess.Stamp),
	}

	vals := log.MakeValues("session", hex.EncodeToString(req.Session.Id))
	log.Info(ctx, vals, "Sending session to peer")

	client := api.NewReplicationClient(c.conn)
	_, err := client.SetSession(ctx, &req)
	if err != nil {
		log.Error(ctx, err, "Failed sending session to peer")
		go c.reestablish(ctx, self, err)
	}
}

func (c *connection) streamSessions(ctx context.Context, self *Self) {
	log.Info(ctx, nil, "Streaming new sessions from peer")

	req := api.StreamSessionsRequest{
		PeerId: self.id,
		From:   latestToProto(self.copyLatest()),
	}

	client := api.NewReplicationClient(c.conn)
	stream, err := client.StreamSessions(ctx, &req)
	if err != nil {
		log.Error(ctx, err, "Failed streaming sessions from peer")
		go c.reestablish(ctx, self, err)
		return
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Error(ctx, err, "Failed receiving session from peer")
			go c.reestablish(ctx, self, err)
			return
		}

		vals := log.MakeValues("session", hex.EncodeToString(res.Session.Id))
		log.Info(ctx, vals, "Received session from peer")

		sess := state.StampedSession{
			Session: sessionFromProto(res.Session),
			Stamp:   stampFromProto(res.Stamp),
		}
		_, ok := self.sessStore.SetSession(sess)
		if !ok {
			log.Error(ctx, nil, "Received session out of order from peer")
			go c.reestablish(ctx, self, err)
			return
		}
	}

	c.live = true
	ch := c.init
	if ch != nil {
		close(ch)
	}
}

func (c *connection) disconnect() {
	if c != nil {
		_ = c.conn.Close()
	}
}

func (c *connection) logConnectionState(ctx context.Context) {
	for cont := true; cont; {
		s := c.conn.GetState()
		str := strings.ToLower(strings.Replace(s.String(), "_", " ", -1))
		vals := log.MakeValues("state", str)
		log.Info(ctx, vals, "Connection state changed")
		cont = c.conn.WaitForStateChange(ctx, s)
	}
}
