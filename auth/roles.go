package auth

import "github.com/KnowitSolutions/istio-oidc/state/accesspolicy"

func hasRoles(required accesspolicy.Roles, provided accesspolicy.Roles) bool {
	found := make(map[string]bool, len(provided))
	for _, k := range provided {
		found[k] = true
	}

	allow := true
	for _, v := range required {
		if !found[v] {
			allow = false
			break
		}
	}

	return allow
}
