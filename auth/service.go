package auth

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"istio-keycloak/config"
)

type service struct {
	*config.Service
	oauth2Config oauth2.Config
	oidcProvider *oidc.Provider
	oidcVerifier *oidc.IDTokenVerifier
}

func newService(keycloak string, cfg *config.Service) (*service, error) {
	iss := fmt.Sprintf("%s/auth/realms/%s", keycloak, cfg.Realm)
	prov, err := oidc.NewProvider(context.TODO(), iss) // TODO
	if err != nil {
		log.WithField("issuer", iss).WithError(err).
			Error("Unable to fetch OIDC provider config")
		return nil, err
	}

	oauth2cfg := oauth2.Config{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Endpoint:     prov.Endpoint(),
	}
	verifier := prov.Verifier(&oidc.Config{ClientID: cfg.OIDC.ClientID})

	return &service{
		Service:      cfg,
		oauth2Config: oauth2cfg,
		oidcProvider: prov,
		oidcVerifier: verifier,
	}, nil
}
