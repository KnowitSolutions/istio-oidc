package state

import (
	"crypto/rand"
	"crypto/sha512"
	"sync"
)

type KeyStore interface {
	GetKey() []byte
	MakeKey() ([]byte, error)
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

func (ks *keyStore) MakeKey() ([]byte, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.key = make([]byte, sha512.Size)
	_, err := rand.Read(ks.key)

	return ks.key, err
}
