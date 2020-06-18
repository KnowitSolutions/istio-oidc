package auth

import (
	"context"
	"github.com/apex/log"
	"istio-keycloak/config"
	"sync"
)

type PolicyStore interface {
	getAccessPolicy(string) (*accessPolicy, bool)
	UpdateAccessPolicies(context.Context, []*config.AccessPolicy)
}

type policyStoreImpl struct {
	policies   map[string]*accessPolicy
	policiesMu sync.RWMutex
	mark       bool
	marks      map[string]bool
	updateMu   sync.Mutex
}

func NewPolicyStore() PolicyStore {
	return &policyStoreImpl{
		policies: map[string]*accessPolicy{},
		marks: map[string]bool{},
	}
}

func (ps *policyStoreImpl) getAccessPolicy(name string) (*accessPolicy, bool) {
	ps.policiesMu.RLock()
	defer ps.policiesMu.RUnlock()
	pol, ok := ps.policies[name]
	return pol, ok
}

func (ps *policyStoreImpl) UpdateAccessPolicies(ctx context.Context, pols []*config.AccessPolicy) {
	go ps.updatePolicies(ctx, pols)
}

func (ps *policyStoreImpl) updatePolicies(ctx context.Context, pols []*config.AccessPolicy) {
	var wg sync.WaitGroup
	ps.updateMu.Lock()

	for _, pol := range pols {
		wg.Add(1)
		go ps.updatePolicy(ctx, pol, &wg)
	}

	wg.Wait()

	for k, v := range ps.marks {
		if v != ps.mark {
			delete(ps.marks, k)
			delete(ps.policies, k)
		}
	}

	ps.mark = !ps.mark
	ps.updateMu.Unlock()
}

func (ps *policyStoreImpl) updatePolicy(ctx context.Context, cfg *config.AccessPolicy, wg *sync.WaitGroup) {
	pol, err := newAccessPolicy(ctx, KeycloakURL, cfg)
	if err != nil {
		log.WithError(err).Error("Unable to load access policy")
	}

	ps.policiesMu.Lock()
	ps.policies[cfg.Name] = pol
	ps.policiesMu.Unlock()
	ps.marks[cfg.Name] = ps.mark

	wg.Done()
}
