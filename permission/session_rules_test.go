package permission

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSessionRules_AllowTool(t *testing.T) {
	rules := NewSessionRules()
	require.False(t, rules.Allows("run_bash"))
	rules.AllowTool("run_bash")
	require.True(t, rules.Allows("run_bash"))
	require.False(t, rules.Allows("task"))
}

func TestSessionRules_AllowAll(t *testing.T) {
	rules := NewSessionRules()
	rules.AllowAll()
	require.True(t, rules.Allows("run_bash"))
	require.True(t, rules.Allows("task"))
	require.True(t, rules.Allows("anything"))
}

func TestSessionRules_Clear(t *testing.T) {
	rules := NewSessionRules()
	rules.AllowAll()
	rules.Clear()
	require.False(t, rules.Allows("run_bash"))
	rules.AllowTool("task")
	require.True(t, rules.Allows("task"))
	rules.Clear()
	require.False(t, rules.Allows("task"))
}

func TestChain_ClearSessionRules(t *testing.T) {
	chain := NewChain()
	chain.SessionRules().AllowTool("run_bash")
	require.True(t, chain.SessionRules().Allows("run_bash"))
	chain.ClearSessionRules()
	require.False(t, chain.SessionRules().Allows("run_bash"))
}

func TestSessionRules_NilSafe(t *testing.T) {
	var rules *SessionRules
	require.False(t, rules.Allows("run_bash"))
	rules.AllowTool("run_bash")
	rules.AllowAll()
	rules.Clear()
}
