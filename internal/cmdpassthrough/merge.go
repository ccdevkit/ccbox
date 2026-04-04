package cmdpassthrough

// Merge concatenates multiple passthrough command slices into a single
// deduplicated list, preserving first-seen order.
func Merge(sources ...[]string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, src := range sources {
		for _, s := range src {
			if _, ok := seen[s]; !ok {
				seen[s] = struct{}{}
				result = append(result, s)
			}
		}
	}
	if result == nil {
		result = []string{}
	}
	return result
}
