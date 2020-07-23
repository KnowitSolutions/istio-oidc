package peers

import "sync"

type versions struct {
	vers map[string]uint64
	mu   sync.RWMutex
}

type StampedSession struct {
	*Session
	Version
}

type Version struct {
	PeerId string
	Serial uint64
}

func newVersions() versions {
	return versions{vers: make(map[string]uint64)}
}

func (s *syncer) Stamp(sess *Session) StampedSession {
	s.inc(s.id)
	ver := Version{s.id, s.ver(s.id)}
	return StampedSession{Session: sess, Version: ver}
}

func (v *versions) inc(id string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.vers[id]++
}

func (v *versions) ver(id string) uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.vers[id]
}

func (v *versions) allVers() map[string]uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	vers := make(map[string]uint64, len(v.vers))
	for k, v := range v.vers {
		vers[k] = v
	}

	return vers
}
