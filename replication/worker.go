package replication

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"sync"
	"time"
)

func NewWorker(self *Self, peers *Peers, init chan<- struct{}) {
	go worker(self, peers, init)
}

type closer struct {
	ch   chan<- struct{}
	once sync.Once
}

func (c *closer) close() {
	c.once.Do(func() { close(c.ch) })
}

func worker(self *Self, peers *Peers, init chan<- struct{}) {
	ctx := context.Background()
	tick := time.Tick(config.Replication.EstablishInterval)
	closer := closer{ch: init}

	for {
		refresh(ctx, self, peers, &closer)
		<-tick
	}
}

func refresh(ctx context.Context, self *Self, peers *Peers, closer *closer) {
	log.Info(ctx, nil, "Refreshing peer list")
	eps, err := peers.refresh(ctx)
	if err != nil {
		log.Error(ctx, err, "Failed refreshing peers")
		return
	}

	if len(eps) == 0 {
		closer.close()
	}

	for _, ep := range eps {
		conn, isNew := peers.getConnection(self, ep)
		if isNew {
			conn.cond.L.Lock()
			conn.cond.Wait()
			conn.cond.L.Unlock()
			go track(conn, closer)
		}
	}
}

func track(conn *connection, closer *closer) {
	for {
		if conn.live {
			closer.close()
			return
		} else if conn.dead {
			return
		} else {
			conn.cond.L.Lock()
			conn.cond.Wait()
			conn.cond.L.Unlock()
		}
	}
}
