package replication

import (
	"context"
	"encoding/hex"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/log"
	"github.com/KnowitSolutions/istio-oidc/state"
	"google.golang.org/grpc/codes"
	grpcpeer "google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type Server struct {
	api.UnimplementedReplicationServer
	Self  *Self
	Peers *Peers
}

func (s Server) Handshake(ctx context.Context, req *api.HandshakeRequest) (*api.HandshakeResponse, error) {
	ctx = addressCtx(ctx)
	ctx = log.WithValues(ctx, "peer", req.PeerId)

	if req.PeerId == s.Self.id {
		log.Error(ctx, nil, "Peer ID conflict detected")
		err := status.Error(codes.PermissionDenied, "Peer ID conflict detected")
		return nil, err
	}

	err := s.Peers.refresh(ctx)
	if err != nil {
		log.Error(ctx, nil, "Failed refreshing peers")
		err := status.Error(codes.Unavailable, "Failed refreshing peers")
		return nil, err
	}

	if !s.Peers.hasEp(req.PeerEndpoint) {
		log.Error(ctx, nil, "Unknown peer endpoint")
		err := status.Error(codes.PermissionDenied, "Unknown peer endpoint")
		return nil, err
	}

	s.Peers.addPeer(req.PeerId)
	conn := s.Peers.getConnection(s.Self, req.PeerEndpoint)
	conn.wakeup()
	if conn.live {
		go conn.update(ctx, s.Self, req.Latest)
	}

	return &api.HandshakeResponse{
		PeerId:       s.Self.id,
		PeerEndpoint: s.Self.ep,
		Latest:       latestToProto(s.Self.copyLatest()),
	}, nil
}

func (s Server) SetSession(ctx context.Context, req *api.SetSessionRequest) (*api.SetSessionResponse, error) {
	ctx = addressCtx(ctx)
	ctx = log.WithValues(ctx, "peer", req.PeerId)

	if !s.Peers.hasPeer(req.PeerId) {
		log.Error(ctx, nil, "Unknown peer ID")
		err := status.Error(codes.PermissionDenied, "Unknown peer ID")
		return nil, err
	}

	sess := sessionFromProto(req.Session)
	stamp := stampFromProto(req.Stamp)

	vals := log.MakeValues("session", hex.EncodeToString(req.Session.Id))
	log.Info(ctx, vals, "Received session from peer")

	_, ok := s.Self.sessStore.SetSession(state.StampedSession{Session: sess, Stamp: stamp})
	if ok {
		s.Self.update(stamp.PeerId, stamp.Serial)
		return &api.SetSessionResponse{}, nil
	} else {
		err := status.Error(codes.InvalidArgument, "Detected skipped session from peer")
		return &api.SetSessionResponse{}, err
	}

}

func (s Server) StreamSessions(req *api.StreamSessionsRequest, stream api.Replication_StreamSessionsServer) error {
	ctx := addressCtx(stream.Context())
	ctx = log.WithValues(ctx, "peer", req.PeerId)

	if !s.Peers.hasPeer(req.PeerId) {
		log.Error(ctx, nil, "Unknown peer ID")
		err := status.Error(codes.PermissionDenied, "Unknown peer ID")
		return err
	}

	log.Info(ctx, nil, "Streaming sessions to peer")
	from := stampsFromProto(req.From)
	ch := s.Self.sessStore.StreamSessions(from)

	for e := range ch {
		sess := sessionToProto(e.Session)
		stamp := stampToProto(e.Stamp)

		req := &api.StreamSessionsResponse{
			Session: sess,
			Stamp:   stamp,
		}

		vals := log.MakeValues("session", hex.EncodeToString(req.Session.Id))
		log.Info(ctx, vals, "Sending session to peer")

		err := stream.Send(req)
		if err != nil {
			log.Error(ctx, err, "Failed sending session to peer")
			return err
		}
	}

	return nil
}

func addressCtx(ctx context.Context) context.Context {
	p, _ := grpcpeer.FromContext(ctx)
	addr := p.Addr.String()
	return log.WithValues(ctx, "address", addr)
}
