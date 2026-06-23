package compaction

import "testing"

func TestShouldCompact(t *testing.T) {
	if ShouldCompact(100, 200000, 16384) {
		t.Fatal("should not compact under budget")
	}
	if !ShouldCompact(200000, 200000, 16384) {
		t.Fatal("should compact over budget")
	}
	if !ShouldCompact(183617, 200000, 16384) {
		t.Fatal("should compact when estimate exceeds contextWindow - reserve")
	}
}
