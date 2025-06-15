package search

import (
	"testing"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

func TestFuzzyScoring(t *testing.T) {
	// Simple test to verify fuzzy matching works
	got := fuzzy.MatchNormalizedFold("hello", "hello world")
	if !got {
		t.Errorf("Expected match for hello in hello world")
	}

	// Test score function returns positive for a match
	score := FuzzyMatchScore("Hello World", "hello")
	if score <= 0 {
		t.Errorf("Expected positive score for matching 'hello' in 'Hello World', got %d", score)
	}
}
