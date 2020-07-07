package state

import (
	"context"
	"istio-keycloak/log"
	"istio-keycloak/state/accesspolicy"
	"sync"
)

type AccessPolicyStore interface {
	GetAccessPolicy(string) *accesspolicy.AccessPolicy
	UpdateAccessPolicy(context.Context, *accesspolicy.AccessPolicy)
	DeleteAccessPolicy(context.Context, string)
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
		log.Error(ctx, err, "Error while loading access policy")
	}

	aps.entriesMu.Lock()
	aps.entries[pol.Name] = pol
	aps.entriesMu.Unlock()

	vals := log.MakeValues("AccessPolicy", pol.Name)
	log.Info(ctx, vals,"Updated OIDC settings")
}

func (aps *accessPolicyStore) DeleteAccessPolicy(ctx context.Context, name string) {
	aps.entriesMu.Lock()
	delete(aps.entries, name)
	aps.entriesMu.Unlock()

	vals := log.MakeValues("AccessPolicy", name)
	log.Info(ctx, vals,"Deleted OIDC settings")
}