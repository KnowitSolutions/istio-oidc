package auth

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"istio-keycloak/config"
	"istio-keycloak/logging/errors"
)

type accessPolicy struct {
	*config.AccessPolicy
	oauth2Config oauth2.Config
	oidcProvider *oidc.Provider
	oidcVerifier *oidc.IDTokenVerifier
}

func newAccessPolicy(ctx context.Context, keycloak string, cfg *config.AccessPolicy) (*accessPolicy, error) {
	iss := fmt.Sprintf("%s/auth/realms/%s", keycloak, cfg.Realm)
	prov, err := oidc.NewProvider(ctx, iss)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch OIDC provider config", "issuer", iss)
	}

	oauth2cfg := oauth2.Config{
		ClientID:     cfg.OIDC.ClientID,
		ClientSecret: cfg.OIDC.ClientSecret,
		Endpoint:     prov.Endpoint(),
	}
	verifier := prov.Verifier(&oidc.Config{ClientID: cfg.OIDC.ClientID})

	return &accessPolicy{
		AccessPolicy: cfg,
		oauth2Config: oauth2cfg,
		oidcProvider: prov,
		oidcVerifier: verifier,
	}, nil
}
