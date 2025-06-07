package command

import (
	"strconv"
	"strings"

	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	pkgSearch "github.com/fchastanet/shell-command-bookmarker/pkg/search"
)

// matchFilter returns true if the item with the given ID matches the filter
// value using fuzzy matching.
func matchFilter(filterValue string, cmd *dbmodels.Command) bool {
	if filterValue == "" {
		return true
	}

	// Try to match id
	if id, err := strconv.Atoi(filterValue); err == nil && cmd.GetID() == resource.ID(id) {
		return true
	}

	// Try exact match first (fastest)
	col := cmd.Title + " " + cmd.Description + " " + cmd.Script
	if strings.Contains(strings.ToLower(col), strings.ToLower(filterValue)) {
		return true
	}

	// Try fuzzy subsequence matching if exact match fails
	if pkgSearch.FuzzyMatchSubsequence(col, filterValue) {
		return true
	}

	// Try advanced scoring if needed
	return pkgSearch.FuzzyMatchScore(col, filterValue) > pkgSearch.ScoreThreshold
}
