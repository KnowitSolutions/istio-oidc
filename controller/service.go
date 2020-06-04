package controller

import (
	"bytes"
	"encoding/binary"
	"istio-keycloak/config"
	"strings"
)

type service struct {
	*config.Service
	ingress ingress
	vhosts  []string
}

type ingress struct {
	key_      string
	namespace string
	selector  map[string]string
}

func (i *ingress) key() string {
	if i.key_ == "" {
		var b bytes.Buffer
		for k, v := range i.selector {
			_ = binary.Write(&b, binary.LittleEndian, len(k))
			_ = binary.Write(&b, binary.LittleEndian, k)
			_ = binary.Write(&b, binary.LittleEndian, len(v))
			_ = binary.Write(&b, binary.LittleEndian, v)
		}
		i.key_ = b.String()
	}
	return i.key_
}

func (i *ingress) String() string {
	var b strings.Builder
	b.WriteString(i.namespace)
	b.WriteRune('/')
	b.WriteRune('{')

	first := true
	for k, v := range i.selector {
		if first {
			first = false
		} else {
			b.WriteRune(',')
		}

		b.WriteString(k)
		b.WriteRune('=')
		b.WriteString(v)
	}

	b.WriteRune('}')
	return b.String()
}
