package sync

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"istio-keycloak/state"
)

func hello(id string, serials map[string]uint64) *Message {
	hello := Hello{PeerId: id, Serials: serials}
	return &Message{Message: &Message_Hello{Hello: &hello}}
}

func pull(id string, serial uint64) *Message {
	pull := Pull{PeerId: id, Serial: serial}
	return &Message{Message: &Message_Pull{Pull: &pull}}
}

func push(session *Session, serial uint64) *Message {
	push := Push{Session: session, Serial: serial}
	return &Message{Message: &Message_Push{Push: &push}}
}

func toProto(sess state.Session) *Session {
	return &Session{
		RefreshToken: sess.RefreshToken,
		Expiry:       timestamppb.New(sess.Expiry),
	}
}

func fromProto(sess *Session) state.Session {
	return state.Session{
		RefreshToken: sess.RefreshToken,
		Expiry:       sess.Expiry.AsTime(),
	}
}
