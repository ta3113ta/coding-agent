package spawn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileFor(t *testing.T) {
	p, err := ProfileFor(TypeExplore)
	require.NoError(t, err)
	require.Len(t, p.Tools, 2)

	_, err = ProfileFor(Type("invalid"))
	require.Error(t, err)
}

func TestAllowedTools_GeneralPurposeExcludesTask(t *testing.T) {
	p, err := ProfileFor(TypeGeneralPurpose)
	require.NoError(t, err)
	allowed := AllowedTools(p, []string{"read_file", "run_bash", "task", "write_file"})
	require.False(t, allowed["task"])
	require.True(t, allowed["read_file"])
	require.True(t, allowed["run_bash"])
}

func TestAllowedTools_Explore(t *testing.T) {
	p, _ := ProfileFor(TypeExplore)
	allowed := AllowedTools(p, []string{"read_file", "list_dir", "run_bash"})
	require.True(t, allowed["read_file"])
	require.True(t, allowed["list_dir"])
	require.False(t, allowed["run_bash"])
}
