package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	rbac "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sigs.k8s.io/yaml"
)

//go:generate go run .

func main() {
	dir := root()

	ch := make(chan []byte)
	go res(dir, "api", "crd", ch)
	crds(ch)

	ch = make(chan []byte)
	go res(dir, "controller", "rbac:roleName=istio-oidc", ch)
	roles(ch)
}

func root() string {
	loc, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}

	loc = path.Dir(loc)
	return loc
}

func res(root, where, what string, ch chan<- []byte) {
	cmd := exec.Command("controller-gen", what, "output:stdout")
	cmd.Dir = path.Join(root, where)
	data, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}

	data = bytes.TrimSpace(data)
	data = bytes.TrimPrefix(data, []byte("---\n"))
	re := regexp.MustCompile(`(?m)^---\n`)

	for len(data) > 0 {
		idx := re.FindIndex(data)
		if idx == nil {
			idx = []int{len(data), len(data)}
		}

		ch <- data[:idx[0]]
		data = data[idx[1]:]
	}

	close(ch)
}

func put(file string, res interface{}) {
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		panic(err.Error())
	}

	err = ioutil.WriteFile(file, data, 0644)
	if err != nil {
		panic(err.Error())
	}
}

func crds(ch <-chan []byte) {
	for doc := range ch {
		crd := apiextensions.CustomResourceDefinition{}
		err := yaml.UnmarshalStrict(doc, &crd)
		if err != nil {
			panic(err.Error())
		}

		switch crd.Name {
		case "accesspolicies.krsdev.app":
			put("custom-resource-definition-access-policy.json", crd)
		case "openidproviders.krsdev.app":
			put("custom-resource-definition-openid-provider.json", crd)
		default:
			panic("unknown CRD " + crd.Name)
		}
	}
}

func roles(ch <-chan []byte) {
	for doc := range ch {
		crd := rbac.ClusterRole{}
		err := yaml.UnmarshalStrict(doc, &crd)
		if err != nil {
			panic(err.Error())
		}

		switch crd.Name {
		case "istio-oidc":
			put("cluster-role.json", crd)
		default:
			panic("unknown cluster roles " + crd.Name)
		}
	}
}
