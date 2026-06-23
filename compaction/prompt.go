package compaction

import (
	"fmt"
	"strings"
)

const summaryTemplate = `Summarize the conversation history below for continuation in a coding agent session.
Output a structured summary using exactly these markdown sections:

## Goal
## Constraints & Preferences
## Progress (Done / In Progress / Blocked)
## Key Decisions
## Next Steps
## Critical Context

Preserve enough detail to continue work without re-asking the user. Include files read or modified, errors and fixes, and pending tasks.
Output only the summary sections, no preamble.`

func BuildSummaryPrompt(serialized string, previousSummary string, fileOps FileOps, customInstructions string) string {
	var b strings.Builder
	b.WriteString(summaryTemplate)

	if strings.TrimSpace(previousSummary) != "" {
		b.WriteString("\n\n---\n\nPrevious summary (incorporate and update):\n\n")
		b.WriteString(strings.TrimSpace(previousSummary))
	}

	if len(fileOps.ReadFiles) > 0 || len(fileOps.ModifiedFiles) > 0 {
		b.WriteString("\n\n---\n\nKnown file operations:\n")
		if len(fileOps.ReadFiles) > 0 {
			b.WriteString("<read-files>\n")
			for _, p := range fileOps.ReadFiles {
				fmt.Fprintf(&b, "- %s\n", p)
			}
			b.WriteString("</read-files>\n")
		}
		if len(fileOps.ModifiedFiles) > 0 {
			b.WriteString("<modified-files>\n")
			for _, p := range fileOps.ModifiedFiles {
				fmt.Fprintf(&b, "- %s\n", p)
			}
			b.WriteString("</modified-files>\n")
		}
	}

	if strings.TrimSpace(customInstructions) != "" {
		b.WriteString("\n\n---\n\nFocus instructions:\n")
		b.WriteString(strings.TrimSpace(customInstructions))
	}

	b.WriteString("\n\n---\n\nConversation to summarize:\n\n")
	b.WriteString(serialized)
	return b.String()
}
