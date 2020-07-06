package auth

import "istio-keycloak/state/accesspolicy"

func hasRoles(required accesspolicy.RoleSet, provided accesspolicy.Roles) bool {
	found := make(map[string]bool, len(provided))
	for _, k := range provided {
		found[k] = true
	}

	allow := false
	for _, vs := range required {
		all := true
		for _, v := range vs {
			if !found[v] {
				all = false
				break
			}
		}
		if all {
			allow = true
			break
		}
	}

	return allow
}
