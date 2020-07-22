package sync

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"istio-keycloak/log"
	"istio-keycloak/log/errors"
	"istio-keycloak/state"
	"os"
	"reflect"
	"sync"
)

//go:generate go get google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate go get google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. service.proto

type Syncer interface {
	SynchronizeServer
	Stamp() Version
	Sync(state.Session)
}

type Version struct {
	PeerId string
	Serial uint64
}

type PushFunc func(state.Session, Version)
type PullFunc func(Version) <-chan state.Session
type syncer struct {
	UnimplementedSynchronizeServer
	peers *peers
	push  PushFunc
	pull  PullFunc
	id    string
	vers  map[string]uint64
	mu    sync.RWMutex
}

func NewSyncer(push PushFunc, pull PullFunc) (Syncer, error) {
	id, err := os.Hostname()
	if err != nil {
		err = errors.Wrap(err, "failed making syncer")
		return nil, err
	}

	vers := make(map[string]uint64)
	s := syncer{id: id, vers: vers, push: push, pull: pull}
	s.peers = newPeers(open, s.talk)

	return &s, nil
}

func (s *syncer) Stamp() Version {
	s.inc(s.id)
	return Version{s.id, s.ver(s.id)}
}

func (s *syncer) inc(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vers[id]++
}

func (s *syncer) ver(id string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.vers[id]
}

func (s *syncer) allVers() map[string]uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vers := make(map[string]uint64, len(s.vers))
	for k, v := range s.vers {
		vers[k] = v
	}

	return vers
}

func (s *syncer) Sync(sess state.Session) {
	s.peers.send(push(toProto(sess), s.ver(s.id)))
}

func (s *syncer) Stream(stream Synchronize_StreamServer) error {
	peer, err := s.peers.add(stream)
	if err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}

	s.talk(peer)
	return nil
}

func open(ctx context.Context, conn *grpc.ClientConn) (stream, error) {
	client := NewSynchronizeClient(conn)
	stream, err := client.Stream(ctx)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s *syncer) talk(peer peer) {
	ctx := peer.stream.Context()
	peer.send(hello(s.id, s.allVers()))

	msg := peer.recv()
	if msg == nil {
		err := errors.New("peer didn't send any messages")
		log.Error(ctx, err, "Peering failed")
		return
	}

	hello := msg.Message.(*Message_Hello)
	if hello == nil {
		err := errors.New("peer didn't send hello", "type", reflect.TypeOf(msg).Name())
		log.Error(ctx, err, "Peering failed")
		return
	}

	id := hello.Hello.PeerId
	vers := hello.Hello.Serials
	for id, remote := range vers {
		local := s.ver(id)
		if local < remote {
			peer.send(pull(id, local))
		}
	}

	for msg = peer.recv(); msg != nil; msg = peer.recv() {
		switch msg := msg.Message.(type) {
		case *Message_Pull:
			id := msg.Pull.PeerId
			ver := Version{id, msg.Pull.Serial}

			for sess := range s.pull(ver) {
				s.Sync(sess)
			}

		case *Message_Push:
			sess := fromProto(msg.Push.Session)
			ver := Version{id, msg.Push.Serial}

			s.inc(id)
			s.push(sess, ver)

		default:
			err := errors.New("unexpected message", "type", reflect.TypeOf(msg).Name())
			log.Error(ctx, err, "Peer sent unknown message")
			return
		}
	}
}
