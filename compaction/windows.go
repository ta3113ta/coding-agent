package compaction

const defaultContextWindow = 200000

var modelContextWindows = map[string]int{
	"claude-sonnet-4-5":           200000,
	"claude-3-5-sonnet-20241022":  200000,
	"claude-3-5-sonnet-latest":    200000,
	"anthropic/claude-sonnet-4":   200000,
	"anthropic/claude-sonnet-4-5": 200000,
}

func ContextWindowForModel(model string, override int) int {
	if override > 0 {
		return override
	}
	if w, ok := modelContextWindows[model]; ok {
		return w
	}
	return defaultContextWindow
}
