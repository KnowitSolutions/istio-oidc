package replication

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"time"
)

func NewWorker(self *Self, peers *Peers, init chan<- struct{}) {
	go worker(self, peers, init)
}

func worker(self *Self, peers *Peers, init chan<- struct{}) {
	ctx := context.Background()
	tick := time.Tick(config.Replication.EstablishInterval)

	for {
		success := refresh(ctx, self, peers)
		if success && init != nil {
			close(init)
			init = nil
		}
		<-tick
	}
}

func refresh(ctx context.Context, self *Self, peers *Peers) bool {
	log.Info(ctx, nil, "Refreshing peer list")
	err := peers.refresh(ctx)
	if err != nil {
		log.Error(ctx, err, "Failed refreshing peers")
		return false
	}

	eps := peers.getEps()
	joined := len(eps) == 0

	for _, ep := range eps {
		conn := peers.getConnection(self, ep)
		conn.wait()
		joined = joined || conn.live
	}

	return joined
}
