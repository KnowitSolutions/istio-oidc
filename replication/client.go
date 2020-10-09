package replication

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/state/session"
)

type Client struct {
	Self  *Self
	Peers *Peers
}

func (c Client) SetSession(sess session.Stamped) {
	ctx := context.Background()
	c.Self.update(c.Self.id, sess.Serial)

	conns := c.Peers.getConnections()
	for _, conn := range conns {
		go conn.setSession(ctx, c.Self, sess)
	}
}