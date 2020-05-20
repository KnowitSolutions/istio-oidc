package config

type Service struct {
	Name string
	Realm string
	OIDC OIDC
	// TODO: Resources
}

type OIDC struct {
	ClientID string
	ClientSecret string
	CallbackPath string
}

// TODO: Remember to log all errors here
func (cfg *Service) Validate() error {
	return nil
}
