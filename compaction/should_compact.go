package compaction

func ShouldCompact(estimate, contextWindow, reserveTokens int) bool {
	if contextWindow <= 0 {
		return false
	}
	budget := contextWindow - reserveTokens
	if budget < 0 {
		budget = 0
	}
	return estimate > budget
}
