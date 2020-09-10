package replication

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/state"
)

type Client struct {
	Self  *Self
	Peers *Peers
}

func (c Client) SetSession(sess state.StampedSession) {
	ctx := context.Background()
	c.Self.update(c.Self.id, sess.Serial)

	conns := c.Peers.getConnections()
	for _, conn := range conns {
		go conn.setSession(ctx, c.Self, sess)
	}
}