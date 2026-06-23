package compaction

import (
	"coding-agent/types"
	"slices"
)

func messageChars(m types.Message) int {
	n := len(m.Content)
	for _, tc := range m.ToolCalls {
		n += len(tc.Input) + len(tc.Name)
	}
	return n
}

// SplitMessagesByTokens divides history into prefix (to summarize) and suffix (to keep).
// Walks backward from the newest message until keepRecentTokens is reached, then aligns
// the split point so tool call groups are not broken.
func SplitMessagesByTokens(msgs []types.Message, keepRecentTokens int) (prefix, suffix []types.Message) {
	if len(msgs) == 0 || keepRecentTokens <= 0 {
		return nil, msgs
	}

	acc := 0
	splitAt := len(msgs)
	for i, val := range slices.Backward(msgs) {
		acc += messageChars(val)
		if acc/4 >= keepRecentTokens {
			splitAt = i
			break
		}
	}
	if splitAt >= len(msgs) {
		return nil, msgs
	}

	splitAt = alignSplitPoint(msgs, splitAt)
	if splitAt <= 0 {
		return nil, msgs
	}
	return msgs[:splitAt], msgs[splitAt:]
}
