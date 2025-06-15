package search

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
)

// Constants for fuzzy matching
const (
	// MaxScore is the maximum score for fuzzy matching
	MaxScore = 100
	// ScoreThreshold is the minimum score considered a match
	ScoreThreshold = 25
)

// FuzzyMatchScore returns a score for how well the pattern matches the text.
// Higher score means better match. Returns -1 if there is no match.
func FuzzyMatchScore(text, pattern string) int {
	if pattern == "" {
		return MaxScore // Empty pattern matches anything with highest score
	}
	if text == "" {
		return -1 // Empty text can't match any pattern
	}

	// Use the rank function from the fuzzy library to get the score
	distance := fuzzy.RankMatchNormalizedFold(pattern, text)
	if distance == -1 {
		return -1 // No match
	}

	// Convert the distance to a score (0-100)
	// Lower distance = better match = higher score
	score := MaxScore - distance
	if score < 0 {
		score = 0
	}

	return score
}
