package replication

import (
	"context"
	"encoding/hex"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/state/session"
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
	ep   string
	conn *grpc.ClientConn

	live bool
	dead bool

	once sync.Once
	cond sync.Cond
}

func newConnection(self *Self, peer string, authority string) *connection {
	conn := connection{ep: peer, cond: *sync.NewCond(&sync.Mutex{})}
	conn.conn, _ = grpc.Dial(peer, grpc.WithInsecure(), grpc.WithAuthority(authority))

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
	go c.update(ctx, self, res.Latest)
}

func (c *connection) reestablish(ctx context.Context, self *Self, err error) {
	var cont bool
	c.once.Do(func() { cont = true })
	if !cont {
		return
	}

	c.live = false
	c.cond.Broadcast()

	if status.Code(err) == codes.Canceled {
		log.Info(ctx, nil, "Disconnected from peer")
		return
	}

	log.Info(ctx, nil, "Waiting for transport to recover")
	for c.conn.GetState() == connectivity.TransientFailure {
		c.conn.WaitForStateChange(ctx, connectivity.TransientFailure)
		c.conn.WaitForStateChange(ctx, connectivity.Connecting)
	}

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
		c.live = c.streamSessions(ctx, self)
	} else {
		log.Info(ctx, nil, "Peer reports no new sessions")
		c.live = true
	}

	c.cond.Broadcast()
}

func (c *connection) setSession(ctx context.Context, self *Self, sess session.Stamped) {
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

func (c *connection) streamSessions(ctx context.Context, self *Self) bool {
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
		return false
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Error(ctx, err, "Failed receiving session from peer")
			go c.reestablish(ctx, self, err)
			return false
		}

		vals := log.MakeValues("session", hex.EncodeToString(res.Session.Id))
		log.Info(ctx, vals, "Received session from peer")

		sess := session.Stamped{
			Session: sessionFromProto(res.Session),
			Stamp:   stampFromProto(res.Stamp),
		}
		_, err = self.sessStore.Set(sess)
		if err != nil {
			log.Error(ctx, err, "Error setting session")
			go c.reestablish(ctx, self, err)
			return false
		}
	}

	return true
}

func (c *connection) disconnect() {
	c.dead = true
	_ = c.conn.Close()
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
