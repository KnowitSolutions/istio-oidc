package replication

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/state"
)

type Client struct {
	Self  *Self
	Peers *Peers
}

func (c Client) SetSession(ctx context.Context, sess state.StampedSession) {
	c.Self.update(c.Self.id, sess.Serial)

	conns := c.Peers.getPeers()
	for _, conn := range conns {
		go conn.setSession(ctx, c.Self, sess)
	}
}