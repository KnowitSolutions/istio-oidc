package session

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

type Stamped struct {
	Session
	Stamp
}

type Store interface {
	Get(string) (Session, bool)
	Set(Stamped) (Stamped, bool)
	Stream(map[string]uint64) <-chan Stamped
}

type sessionStore struct {
	id   string
	curr uint64

	lookup map[string]Session
	store  map[string]*list.List
	mu     sync.RWMutex
	delMu  sync.RWMutex
}

func NewSessionStore(peerId string) (Store, error) {
	ss := &sessionStore{
		id: peerId,

		lookup: map[string]Session{},
		store:  map[string]*list.List{},
	}

	go ss.cleaner()
	return ss, nil
}

func (ss *sessionStore) Get(id string) (Session, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	sess, ok := ss.lookup[id]
	return sess, ok
}

func (ss *sessionStore) Set(sess Stamped) (Stamped, bool) {
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
		curr := last.Value.(Stamped).Serial + 1
		if sess.Serial != curr {
			return Stamped{}, false
		}
	}

	ss.store[sess.Stamp.PeerId].PushBack(sess)
	ss.lookup[sess.Id] = sess.Session

	return sess, true
}

func (ss *sessionStore) Stream(from map[string]uint64) <-chan Stamped {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	store := make(map[string]*list.List, len(ss.store))
	for k, v := range ss.store {
		store[k] = v
	}

	ch := make(chan Stamped)
	go func() {
		ss.delMu.RLock()
		defer ss.delMu.RUnlock()

		for k, l := range store {
			for e := l.Front(); e != nil; e = e.Next() {
				v := e.Value.(Stamped)
				if v.Serial >= from[k] {
					ch <- v
				}
			}
		}

		close(ch)
	}()
	return ch
}

func (ss *sessionStore) cleaner() {
	tick := time.Tick(config.Sessions.CleaningInterval)
	for {
		<-tick

		min := time.Now().Add(-config.Sessions.CleaningGracePeriod)
		vals := log.MakeValues("min", min.Format(time.RFC3339))
		log.Info(nil, vals, "Cleaning sessions")

		ss.clean(min)
	}
}

func (ss *sessionStore) clean(min time.Time) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	ss.delMu.Lock()
	defer ss.delMu.Unlock()

	for _, l := range ss.store {
		for e := l.Front(); e != nil; e = e.Next() {
			v := e.Value.(Stamped)
			if v.Expiry.Before(min) {
				ss.mu.RUnlock()
				ss.delete(l, e)
				ss.mu.RLock()
			}
		}
	}
}

func (ss *sessionStore) delete(l *list.List, e *list.Element) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	v := e.Value.(Stamped)
	l.Remove(e)
	delete(ss.lookup, v.Id)
}
