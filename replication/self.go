package replication

import (
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/state"
	"sync"
)

type Self struct {
	id string
	ep string

	latest map[string]uint64
	mu     sync.RWMutex

	sessStore state.SessionStore
}

func NewSelf(id string, sessStore state.SessionStore) *Self {
	return &Self{
		id:        id,
		ep:        config.Replication.AdvertiseAddress,
		latest:    map[string]uint64{},
		sessStore: sessStore,
	}
}

func (s *Self) copyLatest() map[string]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	latest := make(map[string]uint64, len(s.latest))
	for k, v := range s.latest {
		latest[k] = v
	}
	return latest
}

func (s *Self) needsUpdate(latest map[string]uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	needed := false
	for k, v := range latest {
		if s.latest[k] < v {
			needed = true
			break
		}
	}
	return needed
}

func (s *Self) update(id string, serial uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.latest[id] < serial {
		s.latest[id] = serial
	}
}
