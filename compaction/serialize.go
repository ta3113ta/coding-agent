package compaction

import (
	"fmt"
	"strings"

	"coding-agent/types"
)

const defaultToolResultMaxLen = 2000

type SerializeOptions struct {
	ToolResultMaxLen int
}

func SerializeConversation(msgs []types.Message, opts SerializeOptions) string {
	maxLen := opts.ToolResultMaxLen
	if maxLen <= 0 {
		maxLen = defaultToolResultMaxLen
	}

	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case "user":
			b.WriteString("[User]: ")
			b.WriteString(m.Content)
		case "assistant":
			b.WriteString("[Assistant]: ")
			b.WriteString(m.Content)
			if len(m.ToolCalls) > 0 {
				b.WriteString("\n[Assistant tool calls]: ")
				for i, tc := range m.ToolCalls {
					if i > 0 {
						b.WriteString("; ")
					}
					fmt.Fprintf(&b, "%s(%s)", tc.Name, string(tc.Input))
				}
			}
		case "tool":
			b.WriteString("[Tool result]")
			if m.IsError {
				b.WriteString(" (error)")
			}
			b.WriteString(": ")
			b.WriteString(truncateToolResult(m.Content, maxLen))
		}
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func truncateToolResult(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	truncated := maxLen - len("... [truncated N chars]")
	if truncated < 0 {
		truncated = 0
	}
	omitted := len(content) - truncated
	return content[:truncated] + fmt.Sprintf("... [truncated %d chars]", omitted)
}
