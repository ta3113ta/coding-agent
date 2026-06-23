package compaction

import (
	"fmt"
	"strings"

	"coding-agent/session"
	"coding-agent/types"
)

const CompactedPrefix = "[Context compacted — earlier conversation summarized]"

func BuildCompactedMessage(summary string) types.Message {
	return types.Message{
		Role: "user",
		Content: fmt.Sprintf("%s\n\n%s\n\nContinue from the recent messages below.",
			CompactedPrefix, strings.TrimSpace(summary)),
	}
}

func ProjectMessages(archive []types.Message, compactions []session.CompactionRecord) []types.Message {
	if len(compactions) == 0 {
		return archive
	}
	latest := compactions[len(compactions)-1]
	if latest.FirstKeptIndex >= len(archive) {
		return []types.Message{BuildCompactedMessage(latest.Summary)}
	}
	suffix := archive[latest.FirstKeptIndex:]
	return append([]types.Message{BuildCompactedMessage(latest.Summary)}, suffix...)
}
