package state

import (
	"crypto/sha512"
	"github.com/apex/log"
	"istio-keycloak/config"
	"sync"
	"time"
)

type Session struct {
	RefreshToken string
	Expiry       time.Time
}

type SessionStore interface {
	Start()
	GetSession(string) (*Session, bool)
	SetSession(string, *Session)
}

type sessionStore struct {
	store map[[sha512.Size]byte]Session
	mu    sync.RWMutex
}

func NewSessionStore() SessionStore {
	return &sessionStore{
		store: map[[sha512.Size]byte]Session{},
	}
}

func (ss *sessionStore) Start() {
	go ss.sessionCleaner()
}

func (ss *sessionStore) GetSession(token string) (*Session, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	hash := sha512.Sum512([]byte(token))
	session, ok := ss.store[hash]
	return &session, ok
}

func (ss *sessionStore) SetSession(token string, session *Session) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	hash := sha512.Sum512([]byte(token))
	ss.store[hash] = *session
}

func (ss *sessionStore) sessionCleaner() {
	tick := time.NewTicker(config.Sessions.CleaningInterval)

	for {
		<-tick.C

		start := time.Now()
		max := start.Add(-config.Sessions.CleaningGracePeriod)
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
