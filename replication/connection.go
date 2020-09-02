package replication

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/state"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"os"
	"sync"
	"time"
)

type connection struct {
	id, ep   string
	conn     *grpc.ClientConn
	globalMu *sync.Mutex
	localMu  sync.Mutex
}

func newConnection(ctx context.Context, self *Self, peer string, mu *sync.Mutex) *connection {
	ctx = log.WithValues(ctx, "peer", peer)

	conn := connection{ep: peer, globalMu: mu}
	conn.conn, _ = grpc.Dial(peer)
	go conn.handshake(ctx, self)

	return &conn
}

func (c *connection) handshake(ctx context.Context, self *Self) {
	c.globalMu.Lock()
	defer c.globalMu.Unlock()
	c.localMu.Lock()
	defer c.localMu.Unlock()

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

	c.id = res.PeerId
	c.update(ctx, self, res.Latest)
}

func (c *connection) reestablish(ctx context.Context, self *Self, err error) {
	if status.Code(err) == codes.Canceled {
		log.Info(ctx, nil, "Disconnected from peer")
	} else {
		log.Info(ctx, nil, "Backing off")
		time.Sleep(config.Replication.ReestablishGracePeriod)
		c.handshake(ctx, self)
	}
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
	c.localMu.Lock()
	defer c.localMu.Unlock()

	req := api.SetSessionRequest{
		PeerId:  self.id,
		Session: sessionToProto(sess.Session),
		Stamp:   stampToProto(sess.Stamp),
	}

	log.Info(ctx, nil, "Setting session on peer")

	client := api.NewReplicationClient(c.conn)
	_, err := client.SetSession(ctx, &req)
	if err != nil {
		log.Error(ctx, err, "Failed setting session on peer")
		go c.reestablish(ctx, self, err)
	}
}

func (c *connection) streamSessions(ctx context.Context, self *Self) {
	c.localMu.Lock()
	defer c.localMu.Unlock()

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
			log.Error(ctx, err, "Failed streaming sessions from peer")
			go c.reestablish(ctx, self, err)
			return
		}

		sess := state.StampedSession{
			Session: sessionFromProto(res.Session),
			Stamp:   stampFromProto(res.Stamp),
		}
		self.sessStore.SetSession(sess)
	}
}
