package sync

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate go get google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate go get google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go:generate protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. service.proto

type syncer struct {
	UnimplementedSynchronizeServer
	peers peers
}

func newSyncer() *syncer {
	s := syncer{}
	s.peers = newPeers()
	s.peers.connect = s.connect
	s.peers.listen = s.listen
	return &s
}

func (s *syncer) connect(ctx context.Context, conn *grpc.ClientConn) (stream, error) {
	client := NewSynchronizeClient(conn)
	stream, err := client.Stream(ctx)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (s *syncer) listen(peer peer) {
	for msg := peer.recv(); msg != nil; msg = peer.recv() {
		switch msg.Message.(type) {
		case *Message_UpdateRequest:
		case *Message_Session:
		}
	}
}

func (s *syncer) Stream(stream Synchronize_StreamServer) error {
	peer, err := s.peers.add(stream)
	if err != nil {
		return status.Error(codes.PermissionDenied, err.Error())
	}

	s.listen(peer)
	return nil
}
