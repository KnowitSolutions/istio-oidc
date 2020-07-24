package state

import (
	"container/list"
	"crypto/sha512"
	"github.com/KnowitSolutions/istio-oidc/config"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
	"github.com/KnowitSolutions/istio-oidc/state/peers"
	"google.golang.org/protobuf/types/known/timestamppb"
	"sync"
	"time"
)

type Session struct {
	Hash         [sha512.Size]byte
	RefreshToken string
	Expiry       time.Time
}

type SessionStore interface {
	GetSession([sha512.Size]byte) *Session
	SetSession(Session)
	Server() peers.PeeringServer
}

type sessionStore struct {
	lookup map[[sha512.Size]byte]Session
	store  map[string]*list.List
	mu     sync.RWMutex
	syncer peers.Syncer
}

func NewSessionStore() (SessionStore, error) {
	ss := &sessionStore{
		lookup: map[[sha512.Size]byte]Session{},
		store:  map[string]*list.List{},
	}

	var err error
	ss.syncer, err = peers.NewSyncer(ss.push, ss.pull)
	if err != nil {
		err = errors.Wrap(err, "failed creating session store")
	}

	go ss.cleaner()
	return ss, nil
}

func (ss *sessionStore) GetSession(hash [sha512.Size]byte) *Session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	sess, ok := ss.lookup[hash]
	if ok {
		return &sess
	} else {
		return nil
	}
}

func (ss *sessionStore) SetSession(sess Session) {
	stamped := ss.syncer.Stamp(toProto(sess))

	ss.mu.Lock()
	ss.store[stamped.PeerId].PushBack(stamped)
	ss.lookup[sess.Hash] = sess
	ss.mu.Unlock()

	ss.syncer.Sync(stamped)
}

func (ss *sessionStore) Server() peers.PeeringServer {
	return ss.syncer.Server()
}

func (ss *sessionStore) push(stamped peers.StampedSession) {
	sess := fromProto(stamped.Session)

	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.store[stamped.PeerId].PushBack(stamped)
	ss.lookup[sess.Hash] = sess
}

func (ss *sessionStore) set(id string, sess Session, stamped peers.StampedSession) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	l := ss.store[id]
	ss.lookup[sess.Hash] = sess
	l.PushBack(stamped)
}

func (ss *sessionStore) pull(ver peers.Version) <-chan peers.StampedSession {
	ch := make(chan peers.StampedSession)

	ss.mu.RLock()
	l := ss.store[ver.PeerId]
	ss.mu.RUnlock()

	var e *list.Element
	for e = l.Front(); e != nil; e = e.Next() {
		v := e.Value.(peers.StampedSession)
		if v.Serial >= ver.Serial {
			break
		}
	}

	go func() {
		for ; e != nil; e = e.Next() {
			v := e.Value.(peers.StampedSession)
			ch <- v
		}
		close(ch)
	}()

	return ch
}

func (ss *sessionStore) cleaner() {
	tick := time.NewTicker(config.Sessions.CleaningInterval)
	for {
		<-tick.C

		start := time.Now()
		min := start.Add(-config.Sessions.CleaningGracePeriod)

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
			v := e.Value.(*peers.StampedSession)
			if v.Expiry.AsTime().Before(min) {
				ss.delete(l, e)
			} else {
				break
			}
		}
	}
}

func (ss *sessionStore) delete(l *list.List, e *list.Element) {
	ss.mu.RUnlock()
	defer ss.mu.RLock()
	ss.mu.Lock()
	defer ss.mu.Unlock()

	v := e.Value.(*peers.StampedSession)
	var hash [sha512.Size]byte
	copy(hash[:], v.Hash)

	l.Remove(e)
	delete(ss.lookup, hash)
}

func toProto(sess Session) *peers.Session {
	return &peers.Session{
		Hash:         sess.Hash[:],
		RefreshToken: sess.RefreshToken,
		Expiry:       timestamppb.New(sess.Expiry),
	}
}

func fromProto(sess *peers.Session) Session {
	var hash [sha512.Size]byte
	copy(hash[:], sess.Hash)

	return Session{
		Hash:         hash,
		RefreshToken: sess.RefreshToken,
		Expiry:       sess.Expiry.AsTime(),
	}
}
