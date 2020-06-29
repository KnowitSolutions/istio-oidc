package state

import (
	"context"
	"github.com/apex/log"
	"sync"
)

type OidcCommunicatorStore interface {
	GetOidc(string) (OidcCommunicator, bool)
	UpdateOicd(context.Context, *AccessPolicy)
	DeleteOidc(string)
}

type oidcCommunicatorStoreImpl struct {
	entries   map[string]OidcCommunicator
	entriesMu sync.RWMutex
}

func NewOidcCommunicatorStore() OidcCommunicatorStore {
	return &oidcCommunicatorStoreImpl{
		entries: map[string]OidcCommunicator{},
	}
}

func (oidc *oidcCommunicatorStoreImpl) GetOidc(name string) (OidcCommunicator, bool) {
	oidc.entriesMu.RLock()
	defer oidc.entriesMu.RUnlock()
	pol, ok := oidc.entries[name]
	return pol, ok
}

func (oidc *oidcCommunicatorStoreImpl) UpdateOicd(ctx context.Context, pol *AccessPolicy) {
	go oidc.updateOidc(ctx, pol)
}

func (oidc *oidcCommunicatorStoreImpl) updateOidc(ctx context.Context, pol *AccessPolicy) {
	var err error
	entry, err := newOIDCCommunicator(ctx, pol)
	if err != nil {
		log.WithError(err).Error("Unable to load access policy")
	}

	oidc.entriesMu.Lock()
	oidc.entries[pol.Name] = entry
	oidc.entriesMu.Unlock()

	log.WithField("AccessPolicy", pol.Name).Info("Updated OIDC settings")
}

func (oidc *oidcCommunicatorStoreImpl) DeleteOidc(name string) {
	oidc.entriesMu.Lock()
	delete(oidc.entries, name)
	oidc.entriesMu.Unlock()

	log.WithField("AccessPolicy", name).Info("Deleted OIDC settings")
}
