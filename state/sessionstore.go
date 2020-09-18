package state

import (
	"container/list"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"sync"
	"time"
)

type Session struct {
	Id           string
	RefreshToken string
	Expiry       time.Time
}

type Stamp struct {
	PeerId string
	Serial uint64
}

type StampedSession struct {
	Session
	Stamp
}

type SessionStore interface {
	GetSession(string) (Session, bool)
	SetSession(StampedSession) (StampedSession, bool)
	StreamSessions([]Stamp) <-chan StampedSession
}

type sessionStore struct {
	id   string
	curr uint64

	lookup map[string]Session
	store  map[string]*list.List
	mu     sync.RWMutex
}

func NewSessionStore(peerId string) (SessionStore, error) {
	ss := &sessionStore{
		lookup: map[string]Session{},
		store:  map[string]*list.List{},
		id:     peerId,
	}
	go ss.cleaner()
	return ss, nil
}

func (ss *sessionStore) GetSession(id string) (Session, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	sess, ok := ss.lookup[id]
	return sess, ok
}

func (ss *sessionStore) SetSession(sess StampedSession) (StampedSession, bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if sess.Stamp == (Stamp{}) {
		ss.curr++
		sess.Stamp = Stamp{PeerId: ss.id, Serial: ss.curr}
	}

	if ss.store[sess.Stamp.PeerId] == nil {
		ss.store[sess.Stamp.PeerId] = list.New()
	}

	last := ss.store[sess.Stamp.PeerId].Back()
	if last != nil {
		curr := last.Value.(StampedSession).Serial + 1
		if sess.Serial != curr {
			return StampedSession{}, false
		}
	}

	ss.store[sess.Stamp.PeerId].PushBack(sess)
	ss.lookup[sess.Id] = sess.Session

	return sess, true
}

func (ss *sessionStore) StreamSessions(from []Stamp) <-chan StampedSession {
	ch := make(chan StampedSession)

	go func() {
		for _, s := range from {
			ss.mu.RLock()
			l := ss.store[s.PeerId]
			ss.mu.RUnlock()

			if l == nil {
				continue
			}

			var e *list.Element
			for e = l.Front(); e != nil; e = e.Next() {
				v := e.Value.(StampedSession)
				if v.Serial >= s.Serial {
					ch <- v
				}
			}
		}

		close(ch)
	}()

	return ch
}

func (ss *sessionStore) cleaner() {
	tick := time.NewTicker(config.Sessions.CleaningInterval)
	for {
		<-tick.C

		min := time.Now().Add(-config.Sessions.CleaningGracePeriod)
		vals := log.MakeValues("min", min.Format(time.RFC3339))
		log.Info(nil, vals, "Cleaning sessions")
		ss.clean(min)
	}
}

func (ss *sessionStore) clean(min time.Time) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for _, l := range ss.store {
		for e := l.Front(); e != nil; e.Next() {
			v := e.Value.(StampedSession)
			if v.Expiry.Before(min) {
				ss.delete(l, e)
			}
		}
	}
}

func (ss *sessionStore) delete(l *list.List, e *list.Element) {
	ss.mu.RUnlock()
	defer ss.mu.RLock()
	ss.mu.Lock()
	defer ss.mu.Unlock()

	v := e.Value.(StampedSession)
	l.Remove(e)
	delete(ss.lookup, v.Id)
}
