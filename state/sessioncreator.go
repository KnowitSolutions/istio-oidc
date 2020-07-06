package state

import (
	"golang.org/x/oauth2"
)

// TODO: Probably unnecessary
type SessionCreator interface {
	CreateSession(tok *oauth2.Token) (Session, error)
}

type Header string
type Headers map[string]Header

func newSessionCreator(_ *AccessPolicy) (SessionCreator, error) {
	return sessionCreator(CreateSession), nil
}

type sessionCreator func(tok *oauth2.Token) Session

func (sc sessionCreator) CreateSession(tok *oauth2.Token) (Session, error) {
	return sc(tok), nil
}

func CreateSession(tok *oauth2.Token) Session {
	return Session{
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}
}
