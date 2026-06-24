package skills

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseValid(t *testing.T) {
	content := `---
name: code-review
description: Review code for quality and security.
disable-model-invocation: true
---
# Code Review

Follow these steps.
`
	meta, body, err := Parse([]byte(content))
	require.NoError(t, err)
	require.Equal(t, "code-review", meta.Name)
	require.Equal(t, "Review code for quality and security.", meta.Description)
	require.True(t, meta.DisableModelInvocation)
	require.Contains(t, body, "# Code Review")
}

func TestParseMissingFrontmatter(t *testing.T) {
	_, _, err := Parse([]byte("# No frontmatter"))
	require.Error(t, err)
}

func TestParseMissingName(t *testing.T) {
	content := `---
description: Do something.
---
# Body
`
	_, _, err := Parse([]byte(content))
	require.Error(t, err)
}

func TestParseMissingDescription(t *testing.T) {
	content := `---
name: foo
---
# Body
`
	_, _, err := Parse([]byte(content))
	require.Error(t, err)
}

func TestParseInvalidName(t *testing.T) {
	content := `---
name: Invalid_Name
description: Bad name format.
---
`
	_, _, err := Parse([]byte(content))
	require.Error(t, err)
}

func TestParseFile(t *testing.T) {
	content := `---
name: commit-message
description: Create git commit messages.
---
# Commit
`
	skill, err := ParseFile("/tmp/commit-message/SKILL.md", []byte(content), SourceBundled, "commit-message")
	require.NoError(t, err)
	require.Equal(t, "/tmp/commit-message/SKILL.md", skill.Path)
	require.Equal(t, SourceBundled, skill.Source)
}
