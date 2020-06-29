package accesspolicy

func contains(list []string, target string) bool {
	for i := range list {
		if list[i] == target {
			return true
		}
	}
	return false
}

func remove(list []string, target string) []string {
	next := make([]string, 0, len(list))
	for i := range list {
		if list[i] != target {
			next = append(next, list[i])
		}
	}
	return next
}