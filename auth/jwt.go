package auth

import (
	"github.com/apex/log"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
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

func makeToken(key []byte, claims interface{}, expiry time.Time) string {
	sk := jose.SigningKey{Algorithm: jose.HS512, Key: key}
	signer, err := jose.NewSigner(sk, nil)
	if err != nil {
		log.WithError(err).Fatal("Unable to make JOSE signer")
	}

	tok := jwt.Signed(signer).Claims(claims)

	if !expiry.IsZero() {
		def := jwt.Claims{Expiry: jwt.NewNumericDate(expiry)}
		tok = tok.Claims(def)
	}

	str, err := tok.CompactSerialize()
	if err != nil {
		log.WithError(err).Fatal("Unable to serialize token")
	}

	return str
}

func parseToken(key []byte, tok string, claims expirable) error {
	parsed, err := jwt.ParseSigned(tok)
	if err != nil {
		log.WithField("token", tok).WithError(err).
			Error("Unable to parse JWT")
		return err
	}

	def := &jwt.Claims{}
	err = parsed.Claims(key, def)
	if err != nil {
		log.WithField("token", tok).WithError(err).
			Error("Unable to deserialize default claims")
		return err
	}

	claims.expiryChecker(func() bool {
		expect := jwt.Expected{Time: time.Now()}
		err = def.ValidateWithLeeway(expect, 0)
		return err != nil
	})

	err = parsed.Claims(key, claims)
	if err != nil {
		log.WithField("token", tok).WithError(err).
			Error("Unable to deserialize custom claims")
		return err
	}

	return nil
}
