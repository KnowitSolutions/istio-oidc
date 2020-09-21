package replication

import (
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/state"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sessionToProto(obj state.Session) *api.Session {
	return &api.Session{
		Id:           []byte(obj.Id),
		RefreshToken: obj.RefreshToken,
		Expiry:       timestamppb.New(obj.Expiry),
	}
}

func sessionFromProto(proto *api.Session) state.Session {
	return state.Session{
		Id:           string(proto.Id),
		RefreshToken: proto.RefreshToken,
		Expiry:       proto.Expiry.AsTime(),
	}
}

func stampToProto(obj state.Stamp) *api.Stamp {
	return &api.Stamp{
		PeerId: obj.PeerId,
		Serial: obj.Serial,
	}
}

func stampFromProto(proto *api.Stamp) state.Stamp {
	return state.Stamp{
		PeerId: proto.PeerId,
		Serial: proto.Serial,
	}
}

func latestToProto(dict map[string]uint64) []*api.Stamp {
	proto := make([]*api.Stamp, 0, len(dict))
	for k, v := range dict {
		proto = append(proto, &api.Stamp{PeerId: k, Serial: v})
	}
	return proto
}

func latestFromProto(proto []*api.Stamp) map[string]uint64 {
	dict := make(map[string]uint64, len(proto))
	for _, obj := range proto {
		dict[obj.PeerId] = obj.Serial
	}
	return dict
}
