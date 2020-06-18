package state

import (
	"crypto/sha512"
	"github.com/apex/log"
	"golang.org/x/oauth2"
	"sync"
	"time"
)

var (
	SessionCleaningInterval    time.Duration
	SessionCleaningGracePeriod time.Duration
)

type SessionStore interface {
	Start()
	GetSession(string) (*Session, bool)
	SetSession(string, *oauth2.Token)
}

type sessionStoreImpl struct {
	store map[[sha512.Size]byte]*Session
	mu    sync.RWMutex
}

func NewSessionStore() SessionStore {
	return &sessionStoreImpl{
		store: map[[sha512.Size]byte]*Session{},
	}
}

func (ss *sessionStoreImpl) Start() {
	go ss.sessionCleaner()
}

func (ss *sessionStoreImpl) GetSession(token string) (*Session, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	hash := sha512.Sum512([]byte(token))
	session, ok := ss.store[hash]
	return session, ok
}

func (ss *sessionStoreImpl) SetSession(token string, data *oauth2.Token) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	hash := sha512.Sum512([]byte(token))
	ss.store[hash] = &Session{
		RefreshToken: data.RefreshToken,
		Expiry:       data.Expiry,
	}
}

func (ss *sessionStoreImpl) sessionCleaner() {
	tick := time.NewTicker(SessionCleaningInterval)

	for {
		<-tick.C

		start := time.Now()
		max := start.Add(-SessionCleaningGracePeriod)
		tot := 0

		log.WithField("max", max.Format(time.RFC3339)).
			Info("Cleaning sessions")

		ss.mu.RLock()
		for k, v := range ss.store {
			if v.Expiry.Before(max) {
				ss.mu.RUnlock()
				ss.mu.Lock()

				delete(ss.store, k)
				tot++

				ss.mu.Unlock()
				ss.mu.RLock()
			}
		}
		ss.mu.RUnlock()

		stop := time.Now()
		dur := stop.Sub(start)
		log.WithFields(log.Fields{"duration": dur, "total": tot}).
			Info("Done cleaning sessions")
	}
}
