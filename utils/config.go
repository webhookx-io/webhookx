package utils

func Expand(presets map[string][]string, strs []string) []string {
	var expanded []string
	for _, r := range strs {
		if set, ok := presets[r]; ok {
			expanded = append(expanded, Expand(presets, set)...)
		} else {
			expanded = append(expanded, r)
		}
	}
	return expanded
}
