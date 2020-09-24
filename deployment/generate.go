package main

import (
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"os/exec"
	"path"
)

//go:generate go run .

func main() {
	dir := root()
	fmt.Println("Generating custom resource definition")
	gen("api", "crd", dir, "custom-resource-definition.json")
	fmt.Println("Generating cluster role")
	gen("controller", "rbac:roleName=istio-oidc", dir, "cluster-role.json")
}

func root() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	buf, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}

	loc := string(buf[:len(buf)-1])
	return loc
}

func gen(where, what, root, name string) {
	cmd := exec.Command("controller-gen", what, "output:stdout")
	cmd.Dir = path.Join(root, where)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err.Error())
	}

	err = cmd.Start()
	if err != nil {
		panic(err.Error())
	}

	dec := yaml.NewYAMLToJSONDecoder(stdout)
	doc := make(map[string]interface{})
	_ = dec.Decode(new(interface{}))
	err = dec.Decode(&doc)
	if err != nil {
		panic(err.Error())
	}

	err = cmd.Wait()
	if err != nil {
		panic(err.Error())
	}

	loc := path.Join(root, "deployment", name)
	file, err := os.Create(loc)
	if err != nil {
		panic(err.Error())
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	err = enc.Encode(doc)
	if err != nil {
		panic(err.Error())
	}
}
