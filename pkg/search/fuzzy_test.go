package search

import (
	"testing"
)

func TestFuzzyMatchSubsequence(t *testing.T) {
	tests := []struct {
		text     string
		pattern  string
		expected bool
	}{
		// Empty pattern should match anything
		{"hello", "", true},
		// Empty text shouldn't match non-empty pattern
		{"", "hello", false},

		// Exact matches
		{"hello", "hello", true},
		{"hello", "hell", true},
		{"hello", "ello", true},
		{"hello", "hlo", true},

		// Subsequence matches
		{"hello world", "hlo wld", true},
		{"documentation", "dcmntn", true},
		{"abcdef", "ace", true},
		{"shell command bookmarker", "shllcmdbkm", true},

		// Non-matches
		{"hello", "world", false},
		{"hello", "leh", false},      // Out of order
		{"abcdef", "abcdefg", false}, // Pattern longer than text
	}

	for _, test := range tests {
		result := FuzzyMatchSubsequence(test.text, test.pattern)
		if result != test.expected {
			t.Errorf("FuzzyMatchSubsequence(%q, %q) = %v; expected %v",
				test.text, test.pattern, result, test.expected)
		}
	}
}

func TestCaseInsensitivity(t *testing.T) {
	tests := []struct {
		text     string
		pattern  string
		expected bool
	}{
		{"Hello World", "hello", true},
		{"HELLO", "hello", true},
		{"hello", "HELLO", true},
		{"ShElL CoMmAnD", "shlcmd", true},
	}

	for _, test := range tests {
		result := FuzzyMatchSubsequence(test.text, test.pattern)
		if result != test.expected {
			t.Errorf("Case insensitive FuzzyMatchSubsequence(%q, %q) = %v; expected %v",
				test.text, test.pattern, result, test.expected)
		}
	}
}
