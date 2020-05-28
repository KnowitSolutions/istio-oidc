package config

type Service struct {
	Name string
	Realm string
	OIDC OIDC
	Routes Routes
}

type OIDC struct {
	ClientID string
	ClientSecret string
	CallbackPath string
}

type Routes map[string]Route

type Route struct {
	EnableAuthz bool
	Roles Roles
}

type Roles map[string][]string

// TODO: Remember to log all errors here
func (cfg *Service) Validate() error {
	return nil
}
