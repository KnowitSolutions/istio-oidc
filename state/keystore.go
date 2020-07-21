package state

import (
	"sync"
)

type KeyStore interface {
	GetKey() []byte
	UpdateKey(key []byte)
}

type keyStore struct {
	key []byte
	mu  sync.RWMutex
}

func NewKeyStore() KeyStore {
	return &keyStore{}
}

func (ks *keyStore) GetKey() []byte {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.key
}

func (ks *keyStore) UpdateKey(key []byte) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	ks.key = key
}
