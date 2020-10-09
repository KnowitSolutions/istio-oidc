package auth

func hasRoles(required []string, provided map[string][]string) bool {
	count := 0
	for _, v := range provided {
		count += len(v)
	}

	resolved := make([]string, 0, count)
	for k, v := range provided {
		prefixed := make([]string, len(v))
		for i, v := range v {
			prefixed[i] = k + v
		}
		resolved = append(resolved, prefixed...)
	}

	found := make(map[string]bool, len(provided))
	for _, k := range resolved {
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
