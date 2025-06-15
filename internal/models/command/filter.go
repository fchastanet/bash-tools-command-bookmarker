package command

import (
	"strconv"
	"strings"

	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	pkgSearch "github.com/fchastanet/shell-command-bookmarker/pkg/search"
)

// matchFilter returns true if the item with the given ID matches the filter
// value using fuzzy matching.
func matchFilter(filterValue string, cmd *dbmodels.Command) (matched bool, score int) {
	if filterValue == "" {
		return true, 0
	}

	// Try exact match first (fastest)
	// We check the ID as a string, title, description, and script.
	if cmd.Title == filterValue ||
		cmd.Description == filterValue ||
		cmd.Script == filterValue ||
		strconv.Itoa(int(cmd.GetID())) == filterValue {
		return true, pkgSearch.MaxScore
	}

	// Check if the filter value is a substring of any of the fields
	col := cmd.Title + " " + cmd.Description + " " + cmd.Script
	if strings.Contains(strings.ToLower(col), strings.ToLower(filterValue)) {
		return true, pkgSearch.MaxScore - 1
	}

	// Try advanced scoring if needed
	score = pkgSearch.FuzzyMatchScore(col, filterValue)
	return score > pkgSearch.ScoreThreshold, score
}
