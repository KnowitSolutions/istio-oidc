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
	Stream() <-chan AccessPolicy
}

type store struct {
	dict map[string]*AccessPolicy
	mu   sync.RWMutex
}

func NewAccessPolicyStore() Store {
	return &store{
		dict: map[string]*AccessPolicy{},
	}
}

func (s *store) Get(name string) *AccessPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dict[name]
}

func (s *store) Update(ctx context.Context, pol *AccessPolicy) {
	s.mu.Lock()
	s.dict[pol.Name] = pol
	s.mu.Unlock()

	vals := log.MakeValues("AccessPolicy", pol.Name)
	log.Info(ctx, vals, "Updated OIDC settings")
}

func (s *store) Delete(ctx context.Context, name string) {
	s.mu.Lock()
	delete(s.dict, name)
	s.mu.Unlock()

	vals := log.MakeValues("AccessPolicy", name)
	log.Info(ctx, vals, "Deleted OIDC settings")
}

func (s *store) Stream() <-chan AccessPolicy {
	ch := make(chan AccessPolicy)
	go func() {
		defer close(ch)
		s.mu.RLock()
		defer s.mu.RUnlock()

		for _, v := range s.dict {
			ch <- *v
		}
	}()
	return ch
}