package auth

import (
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/logging/errors"
	"time"
)

type expirable struct {
	jwt.Claims
}

func (e *expirable) isExpired() bool {
	expect := jwt.Expected{Time: time.Now()}
	err := e.ValidateWithLeeway(expect, 0)
	return err != nil
}

func makeToken(key []byte, claims interface{}, expiry time.Time) (string, error) {
	sk := jose.SigningKey{Algorithm: jose.HS512, Key: key}
	signer, _ := jose.NewSigner(sk, nil)
	tok := jwt.Signed(signer).Claims(claims)

	if !expiry.IsZero() {
		def := jwt.Claims{Expiry: jwt.NewNumericDate(expiry)}
		tok = tok.Claims(def)
	}

	str, err := tok.CompactSerialize()
	if err != nil {
		return "", errors.Wrap(err, "failed token serialization")
	}

	return str, nil
}

func parseToken(key []byte, tok string, claims interface{}) error {
	parsed, err := jwt.ParseSigned(tok)
	if err != nil {
		return errors.Wrap(err, "unable to parse JWT", "token", tok)
	}

	err = parsed.Claims(key, claims)
	if err != nil {
		return errors.Wrap(err, "unable to deserialize claims", "token", tok)
	}

	return nil
}
