package config

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"github.com/apex/log"
	"strings"
)

const (
	ServiceKey     = "service"
	RolesKey       = "roles"
	GlobalRouteKey = "*"
	GlobalRoleKey  = ""
)

// TODO: Rename to AccessPolicy
type Service struct {
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
func (cfg *Service) Validate() error {
	return nil
}

// TODO: Replace other encode/decode with these methods
func (r *Roles) Encode() string {
	buf := bytes.NewBuffer(nil)
	b64 := base64.NewEncoder(base64.StdEncoding, buf)
	enc := gob.NewEncoder(b64)
	err := enc.Encode(r)
	if err != nil {
		panic(err)
	}

	err = b64.Close()
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func (r *Roles) Decode(str string) error {
	buf := strings.NewReader(str)
	dec := gob.NewDecoder(base64.NewDecoder(base64.StdEncoding, buf))

	err := dec.Decode(r)
	if err != nil {
		log.WithError(err).Error("Unable to decode roles")
		return err
	} else {
		return nil
	}
}
