package state

import (
	"context"
	"github.com/apex/log"
	"sync"
)

type OidcCommunicatorStore interface {
	GetOIDC(string) (OidcCommunicator, bool)
	UpdateOicds(context.Context, []*AccessPolicy)
}

type odicCommunicatorStoreImpl struct {
	entries   map[string]OidcCommunicator
	entriesMu sync.RWMutex
	mark      bool
	marks     map[string]bool
	updateMu  sync.Mutex
}

func NewOidcCommunicatorStore() OidcCommunicatorStore {
	return &odicCommunicatorStoreImpl{
		entries: map[string]OidcCommunicator{},
		marks:   map[string]bool{},
	}
}

func (oidc *odicCommunicatorStoreImpl) GetOIDC(name string) (OidcCommunicator, bool) {
	oidc.entriesMu.RLock()
	defer oidc.entriesMu.RUnlock()
	pol, ok := oidc.entries[name]
	return pol, ok
}

func (oidc *odicCommunicatorStoreImpl) UpdateOicds(ctx context.Context, pols []*AccessPolicy) {
	go oidc.updateOidcs(ctx, pols)
}

func (oidc *odicCommunicatorStoreImpl) updateOidcs(ctx context.Context, pols []*AccessPolicy) {
	var wg sync.WaitGroup
	oidc.updateMu.Lock()

	for _, pol := range pols {
		wg.Add(1)
		go oidc.updateOidc(ctx, pol, &wg)
	}

	wg.Wait()

	for k, v := range oidc.marks {
		if v == oidc.mark {
			log.WithField("AccessPolicy", k).Info("Updated OIDC settings")
		} else {
			log.WithField("AccessPolicy", k).Info("Deleted OIDC settings")
			delete(oidc.marks, k)
			delete(oidc.entries, k)
		}
	}

	oidc.mark = !oidc.mark
	oidc.updateMu.Unlock()
}

func (oidc *odicCommunicatorStoreImpl) updateOidc(ctx context.Context, pol *AccessPolicy, wg *sync.WaitGroup) {
	var err error
	entry, err := newOIDCCommunicator(ctx, pol)
	if err != nil {
		log.WithError(err).Error("Unable to load access policy")
	}

	oidc.entriesMu.Lock()
	oidc.entries[pol.Name] = entry
	oidc.entriesMu.Unlock()
	oidc.marks[pol.Name] = oidc.mark

	wg.Done()
}
