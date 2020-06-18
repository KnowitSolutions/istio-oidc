package state

import (
	"context"
	"github.com/apex/log"
	"sync"
)

type OidcCommunicatorStore interface {
	GetOIDC(string) (OidcCommunicator, bool)
	UpdateOIDCs(context.Context, []*AccessPolicy)
}

type odicCommunicatorStoreImpl struct {
	entries    map[string]OidcCommunicator
	policiesMu sync.RWMutex
	mark       bool
	marks      map[string]bool
	updateMu   sync.Mutex
}

func NewOidcCommunicatorStore() OidcCommunicatorStore {
	return &odicCommunicatorStoreImpl{
		entries: map[string]OidcCommunicator{},
		marks:   map[string]bool{},
	}
}

func (oidc *odicCommunicatorStoreImpl) GetOIDC(name string) (OidcCommunicator, bool) {
	oidc.policiesMu.RLock()
	defer oidc.policiesMu.RUnlock()
	pol, ok := oidc.entries[name]
	return pol, ok
}

func (oidc *odicCommunicatorStoreImpl) UpdateOIDCs(ctx context.Context, pols []*AccessPolicy) {
	go oidc.updatePolicies(ctx, pols)
}

func (oidc *odicCommunicatorStoreImpl) updatePolicies(ctx context.Context, pols []*AccessPolicy) {
	var wg sync.WaitGroup
	oidc.updateMu.Lock()

	for _, pol := range pols {
		wg.Add(1)
		go oidc.updatePolicy(ctx, pol, &wg)
	}

	wg.Wait()

	for k, v := range oidc.marks {
		if v != oidc.mark {
			delete(oidc.marks, k)
			delete(oidc.entries, k)
		}
	}

	oidc.mark = !oidc.mark
	oidc.updateMu.Unlock()
}

func (oidc *odicCommunicatorStoreImpl) updatePolicy(ctx context.Context, pol *AccessPolicy, wg *sync.WaitGroup) {
	var err error
	entry, err := newOIDCCommunicator(ctx, pol)
	if err != nil {
		log.WithError(err).Error("Unable to load access policy")
	}

	oidc.policiesMu.Lock()
	oidc.entries[pol.Name] = entry
	oidc.policiesMu.Unlock()
	oidc.marks[pol.Name] = oidc.mark

	wg.Done()
}
