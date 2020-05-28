package config

type AccessPolicy struct {
	Gateway string `json:"gateway"`

	Realm string `json:"realm"`
	OIDC  struct {
		CredentialsSecret struct {
			Name            string `json:"name"`
			ClientIDKey     string `json:"clientIDKey"`
			ClientSecretKey string `json:"clientSecretKey"`
		} `json:"credentialsSecretRef"`
		CallbackPath string `json:"callbackPath"`
	} `json:"oidc"`

	GlobalRoleNamespaceKey string `json:"globalRoleNamespaceKey"`
	Routes                 map[string]struct {
		Roles               map[string][]string `json:"roles"`
		DisableAccessPolicy bool                `json:"disableAccessPolicy"`
	} `json:"routes"`
}

// TODO: Defaults:
// AccessPolicy.OIDC.CredentialsSecret.ClientIDKey = "clientID"
// AccessPolicy.OIDC.CredentialsSecret.ClientSecretKey = "clientSecret"
// AccessPolicy.OIDC.CallbackPath = "/odic/callback"
// AccessPolicy.GlobalRoleNamespaceKey = "*"
