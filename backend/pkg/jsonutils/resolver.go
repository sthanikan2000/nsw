package jsonutils

// ResolveTemplate recursively traverses a template structure and replaces string values
// with results from the lookup function. It supports maps and slices.
func ResolveTemplate(template any, lookup func(string) any) any {
	switch v := template.(type) {
	case map[string]any:
		newMap := make(map[string]any)
		for k, val := range v {
			newMap[k] = ResolveTemplate(val, lookup)
		}
		return newMap
	case []any:
		newSlice := make([]any, len(v))
		for i, val := range v {
			newSlice[i] = ResolveTemplate(val, lookup)
		}
		return newSlice
	case string:
		// Treat the string as a path to look up.
		// If the lookup returns nil, we might want to keep the original or return nil.
		// For now, let's return the lookup result.
		if resolved := lookup(v); resolved != nil {
			return resolved
		}
		return v
	default:
		return v
	}
}
