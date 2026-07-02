package compaction

import (
	"context"

	"coding-agent/session"
	"coding-agent/types"
)

type Request struct {
	Archive            []types.Message
	Compactions        []session.CompactionRecord
	SystemPrompt       string
	Model              string
	ContextWindow      int
	Force              bool
	CustomInstructions string
	SessionID          string
}

type Result struct {
	Archive     []types.Message
	Compactions []session.CompactionRecord
	Projected   []types.Message
	Compacted   bool
}

type Compactor interface {
	MaybeCompact(ctx context.Context, req Request) (Result, error)
}

// EstimateTokens returns a rough token count using chars/4 heuristic.
func EstimateTokens(msgs []types.Message) int {
	total := 0
	for _, m := range msgs {
		total += messageChars(m)
	}
	return total / 4
}

// SplitMessages divides history into prefix (to summarize) and suffix (to keep).
// keepRecent is the target number of messages to retain from the end.
// The split never breaks an assistant message with ToolCalls from its tool results.
func SplitMessages(msgs []types.Message, keepRecent int) (prefix, suffix []types.Message) {
	if len(msgs) == 0 || keepRecent <= 0 {
		return nil, msgs
	}
	if keepRecent >= len(msgs) {
		return nil, msgs
	}

	splitAt := len(msgs) - keepRecent
	splitAt = alignSplitPoint(msgs, splitAt)

	if splitAt <= 0 {
		return nil, msgs
	}
	return msgs[:splitAt], msgs[splitAt:]
}

// alignSplitPoint moves splitAt backward so we never split between an
// assistant message with tool calls and its following tool result messages.
func alignSplitPoint(msgs []types.Message, splitAt int) int {
	for splitAt > 0 && splitAt < len(msgs) {
		if msgs[splitAt].Role == "tool" {
			splitAt--
			continue
		}
		if splitAt > 0 && msgs[splitAt-1].Role == "assistant" && len(msgs[splitAt-1].ToolCalls) > 0 {
			splitAt--
			continue
		}
		break
	}
	return splitAt
}
