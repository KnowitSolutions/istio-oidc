package config

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"istio-keycloak/logging/errors"
	"strings"
)

const (
	AccessPolicyKey = "policy"
	RolesKey        = "roles"
	GlobalRouteKey  = "*"
	GlobalRoleKey   = ""
)

// TODO: Rename to AccessPolicy
type AccessPolicy struct {
	Name   string
	Realm  string
	OIDC   OIDC
	Global Route
	Routes Routes
}

type OIDC struct {
	ClientID     string
	ClientSecret string
	CallbackPath string
}

type Routes map[string]Route

type Route struct {
	EnableAuthz bool
	Roles       Roles
}

type Roles map[string][]string

// TODO: Remember to log all errors here
func (cfg *AccessPolicy) Validate() error {
	return nil
}

// TODO: Replace other encode/decode with these methods
func (r *Roles) Encode() (string, error) {
	buf := bytes.NewBuffer(nil)
	b64 := base64.NewEncoder(base64.StdEncoding, buf)
	enc := gob.NewEncoder(b64)
	err := enc.Encode(r)
	_ = b64.Close()
	return buf.String(), err
}

func (r *Roles) Decode(str string) error {
	buf := strings.NewReader(str)
	b64 := base64.NewDecoder(base64.StdEncoding, buf)
	dec := gob.NewDecoder(b64)
	return errors.Wrap(dec.Decode(r), "unable to decode roles")
}
