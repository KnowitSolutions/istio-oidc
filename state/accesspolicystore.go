package state

import (
	"context"
	"github.com/apex/log"
	"sync"
)

type AccessPolicyHelper struct {
	OidcHelper
	SessionCreator
}

type AccessPolicyStore interface {
	GetAccessPolicyHelper(string) AccessPolicyHelper
	UpdateAccessPolicy(context.Context, *AccessPolicy)
	DeleteAccessPolicy(string)
}

type accessPolicyStore struct {
	entries   map[string]AccessPolicyHelper
	entriesMu sync.RWMutex
}

func NewAccessPolicyStore() AccessPolicyStore {
	return &accessPolicyStore{
		entries: map[string]AccessPolicyHelper{},
	}
}

func (aps *accessPolicyStore) GetAccessPolicyHelper(name string) AccessPolicyHelper {
	aps.entriesMu.RLock()
	defer aps.entriesMu.RUnlock()
	return aps.entries[name]
}

func (aps *accessPolicyStore) UpdateAccessPolicy(ctx context.Context, pol *AccessPolicy) {
	go aps.updateAccessPolicy(ctx, pol)
}

func (aps *accessPolicyStore) updateAccessPolicy(ctx context.Context, pol *AccessPolicy) {
	oidc, err := newOidcHelper(ctx, pol)
	if err != nil {
		log.WithError(err).Error("Error while loading access policy")
	}

	header, err := newSessionCreator(pol)
	if err != nil {
		log.WithError(err).Error("Error while loading access policy")
	}

	aps.entriesMu.Lock()
	aps.entries[pol.Name] = AccessPolicyHelper{oidc, header}
	aps.entriesMu.Unlock()

	log.WithField("AccessPolicy", pol.Name).Info("Updated OIDC settings")
}

func (aps *accessPolicyStore) DeleteAccessPolicy(name string) {
	aps.entriesMu.Lock()
	delete(aps.entries, name)
	aps.entriesMu.Unlock()

	log.WithField("AccessPolicy", name).Info("Deleted OIDC settings")
}
