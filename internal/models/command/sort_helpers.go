package command

import (
	"log/slog"
	"strings"

	"github.com/fchastanet/shell-command-bookmarker/internal/models/structure"
	"github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/sort"
)

// compareBySortField compares two commands by the given field
func compareBySortField(i, j *models.Command, field structure.Field) int {
	switch field {
	case structure.FieldID:
		return sort.CompareID(i, j)
	case structure.FieldTitle:
		return strings.Compare(i.Title, j.Title)
	case structure.FieldFilterScore:
		return sort.CompareInt(i.FilterScore, j.FilterScore)
	case structure.FieldScript:
		return strings.Compare(i.Script, j.Script)
	case structure.FieldStatus:
		return strings.Compare(string(i.Status), string(j.Status))
	case structure.FieldLintStatus:
		return strings.Compare(string(i.LintStatus), string(j.LintStatus))
	case structure.FieldCreationDate:
		return sort.CompareTime(i.CreationDatetime, j.CreationDatetime)
	case structure.FieldModificationDate:
		return sort.CompareTime(i.ModificationDatetime, j.ModificationDatetime)
	default:
		slog.Warn("Unknown sort field", "field", field)
		return 0
	}
}
