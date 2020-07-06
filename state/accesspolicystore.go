package state

import (
	"context"
	"github.com/apex/log"
	"istio-keycloak/state/accesspolicy"
	"sync"
)

type AccessPolicyStore interface {
	GetAccessPolicy(string) *accesspolicy.AccessPolicy
	UpdateAccessPolicy(context.Context, *accesspolicy.AccessPolicy)
	DeleteAccessPolicy(string)
}

type accessPolicyStore struct {
	entries   map[string]*accesspolicy.AccessPolicy
	entriesMu sync.RWMutex
}

func NewAccessPolicyStore() AccessPolicyStore {
	return &accessPolicyStore{
		entries: map[string]*accesspolicy.AccessPolicy{},
	}
}

func (aps *accessPolicyStore) GetAccessPolicy(name string) *accesspolicy.AccessPolicy {
	aps.entriesMu.RLock()
	defer aps.entriesMu.RUnlock()
	return aps.entries[name]
}

func (aps *accessPolicyStore) UpdateAccessPolicy(ctx context.Context, pol *accesspolicy.AccessPolicy) {
	go aps.updateAccessPolicy(ctx, pol)
}

func (aps *accessPolicyStore) updateAccessPolicy(ctx context.Context, pol *accesspolicy.AccessPolicy) {
	err := pol.UpdateOidcProvider(ctx)
	if err != nil {
		log.WithError(err).Error("Error while loading access policy")
	}

	aps.entriesMu.Lock()
	aps.entries[pol.Name] = pol
	aps.entriesMu.Unlock()

	log.WithField("AccessPolicy", pol.Name).Info("Updated OIDC settings")
}

func (aps *accessPolicyStore) DeleteAccessPolicy(name string) {
	aps.entriesMu.Lock()
	delete(aps.entries, name)
	aps.entriesMu.Unlock()

	log.WithField("AccessPolicy", name).Info("Deleted OIDC settings")
}
