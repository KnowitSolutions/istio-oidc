// +groupName=krsdev.app
// +versionName=v1
package api

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	GroupVersion  = schema.GroupVersion{Group: "krsdev.app", Version: "v1"}
	schemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = schemeBuilder.AddToScheme
)

func init() {
	schemeBuilder.Register(&AccessPolicy{}, &AccessPolicyList{})
}
