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

type keyStoreImpl struct {
	key []byte
	mu  sync.RWMutex
}

func NewKeyStore() KeyStore {
	return &keyStoreImpl{}
}

func (ks *keyStoreImpl) GetKey() []byte {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.key
}

func (ks *keyStoreImpl) MakeKey() ([]byte, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.key = make([]byte, sha512.Size)
	_, err := rand.Read(ks.key)

	return ks.key, err
}
