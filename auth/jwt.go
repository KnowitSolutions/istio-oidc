package auth

import (
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"istio-keycloak/logging/errors"
	"time"
)

type expirable interface {
	expiryChecker(func() bool)
}

type expirableImpl struct {
	isExpired func() bool
}

func (e *expirableImpl) expiryChecker(f func() bool) {
	e.isExpired = f
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

func parseToken(key []byte, tok string, claims expirable) error {
	parsed, err := jwt.ParseSigned(tok)
	if err != nil {
		return errors.Wrap(err, "unable to parse JWT", "token", tok)
	}

	def := &jwt.Claims{}
	err = parsed.Claims(key, def)
	if err != nil {
		return errors.Wrap(err, "unable to deserialize default claims", "token", tok)
	}

	claims.expiryChecker(func() bool {
		expect := jwt.Expected{Time: time.Now()}
		err = def.ValidateWithLeeway(expect, 0)
		return err != nil
	})

	err = parsed.Claims(key, claims)
	if err != nil {
		return errors.Wrap(err, "unable to deserialize custom claims", "token", tok)
	}

	return nil
}
