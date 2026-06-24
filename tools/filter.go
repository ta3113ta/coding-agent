package tools

// Filter returns a new registry containing only tools whose names are in allowed.
func Filter(src *Registry, allowed map[string]bool) *Registry {
	if src == nil {
		return NewRegistry()
	}
	out := NewRegistry()
	for name, t := range src.tools {
		if allowed[name] {
			out.Register(t)
		}
	}
	return out
}
