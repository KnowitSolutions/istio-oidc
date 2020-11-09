package openidprovider

import (
	"context"
	"github.com/KnowitSolutions/istio-oidc/api"
	"github.com/KnowitSolutions/istio-oidc/log/errors"
)

type openIdProviderSpec api.OpenIDProviderSpec
type openIDProviderRoleMappings []api.OpenIDProviderRoleMapping

func New(ctx context.Context, op *api.OpenIDProvider) (OpenIdProvider, error) {
	spec := openIdProviderSpec(op.Spec)
	return spec.convert(ctx, op.Namespace + "/" + op.Name)
}

func (op openIdProviderSpec) convert(ctx context.Context, name string) (OpenIdProvider, error) {
	addr := op.Issuer + "/.well-known/openid-configuration"

	cfg := openIdConfiguration{}
	err := doJsonRequest(ctx, addr, &cfg)
	if err != nil {
		err = errors.Wrap(err, "unable to fetch OIDC provider config", "issuer", op.Issuer)
		return OpenIdProvider{}, err
	}

	roleMappings := openIDProviderRoleMappings(op.RoleMappings)
	maps, err := roleMappings.convert()
	if err != nil {
		err = errors.Wrap(err, "unable to parse role mappings")
		return OpenIdProvider{}, err
	}

	return OpenIdProvider{Name: name, cfg: cfg, maps: maps}, nil
}

func (oprm openIDProviderRoleMappings) convert() ([]roleMapping, error) {
	maps := make([]roleMapping, len(oprm))
	for i, rm := range oprm {
		from, ok := fromStrToConst[rm.From]
		if !ok {
			err := errors.New("invalid from", "from", rm.From)
			return nil, err
		}

		path, _, err := parseRolePath([]rune(rm.Path))
		if err != nil {
			return nil, err
		}
		
		maps[i] = roleMapping{from, rm.Prefix, path}
	}
	return maps, nil
}
