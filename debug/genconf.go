package main

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"istio-keycloak/config"
)

// Generates roles config string for use in envoy.yaml or envoyfilter.yaml
func main() {
	roles := &config.Roles{
		"": {"global"},
		"test": {"local"},
	}

	buf := bytes.NewBuffer(nil)
	b64 := base64.NewEncoder(base64.StdEncoding, buf)
	enc := gob.NewEncoder(b64)
	err := enc.Encode(roles)
	if err != nil {
		panic(err)
	}

	err = b64.Close()
	if err != nil {
		panic(err)
	}

	print(buf.String())
}
