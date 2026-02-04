package utils

func ResolveAlias(aliasMap map[string][]string, aliases []string) []string {
	var resolved []string

	for _, alias := range aliases {
		if v, ok := aliasMap[alias]; ok {
			resolved = append(resolved, ResolveAlias(aliasMap, v)...)
		} else {
			resolved = append(resolved, alias)
		}
	}

	return resolved
}
