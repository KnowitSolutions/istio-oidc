package accesspolicy

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/log"
	"sync"
)

type Store interface {
	Get(string) *AccessPolicy
	Update(context.Context, *AccessPolicy)
	Delete(context.Context, string)
}

type store struct {
	entries   map[string]*AccessPolicy
	entriesMu sync.RWMutex
}

func NewAccessPolicyStore() Store {
	return &store{
		entries: map[string]*AccessPolicy{},
	}
}

func (s *store) Get(name string) *AccessPolicy {
	s.entriesMu.RLock()
	defer s.entriesMu.RUnlock()
	return s.entries[name]
}

func (s *store) Update(ctx context.Context, pol *AccessPolicy) {
	s.entriesMu.Lock()
	s.entries[pol.Name] = pol
	s.entriesMu.Unlock()

	vals := log.MakeValues("AccessPolicy", pol.Name)
	log.Info(ctx, vals, "Updated OIDC settings")
}

func (s *store) Delete(ctx context.Context, name string) {
	s.entriesMu.Lock()
	delete(s.entries, name)
	s.entriesMu.Unlock()

	vals := log.MakeValues("AccessPolicy", name)
	log.Info(ctx, vals, "Deleted OIDC settings")
}
