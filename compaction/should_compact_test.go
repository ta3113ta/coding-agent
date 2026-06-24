package compaction

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldCompact(t *testing.T) {
	require.False(t, ShouldCompact(100, 200000, 16384))
	require.True(t, ShouldCompact(200000, 200000, 16384))
	require.True(t, ShouldCompact(183617, 200000, 16384))
}
