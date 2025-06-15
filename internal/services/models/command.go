package models

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
)

type CommandStatus string

const (
	CommandStatusImported CommandStatus = "IMPORTED"
	CommandStatusSaved    CommandStatus = "SAVED"
	CommandStatusDeleted  CommandStatus = "DELETED"
	CommandStatusObsolete CommandStatus = "OBSOLETE"
)

type Command struct {
	CreationDatetime     time.Time
	ModificationDatetime time.Time
	Title                string
	Description          string
	Script               string
	Status               CommandStatus
	LintIssues           string
	LintStatus           LintStatus
	lintIssuesParsed     []map[string]any
	ID                   resource.ID
	Elapsed              int
	FilterScore          int
}

type LintStatus string

const (
	LintStatusNotAvailable     LintStatus = "NOT_AVAILABLE"
	LintStatusOK               LintStatus = "OK"
	LintStatusWarning          LintStatus = "WARNING"
	LintStatusError            LintStatus = "ERROR"
	LintStatusShellcheckFailed LintStatus = "SHELLCHECK_FAILED"
)

func NewCommand(
	script string,
	elapsed int,
	timestamp time.Time,
) *Command {
	return &Command{
		ID:                   0,
		Title:                "",
		Description:          "",
		Script:               script,
		Elapsed:              elapsed,
		LintIssues:           "[]",
		lintIssuesParsed:     nil,
		LintStatus:           LintStatusNotAvailable,
		Status:               CommandStatusImported,
		CreationDatetime:     timestamp,
		ModificationDatetime: time.Now(),
		FilterScore:          0,
	}
}

func (c *Command) IsEditable() bool {
	return c.Status == CommandStatusImported ||
		c.Status == CommandStatusSaved
}

// getLintIssues parses the JSON lint issues and returns them as structured data
func (c *Command) GetLintIssues() []map[string]any {
	if c.lintIssuesParsed != nil {
		return c.lintIssuesParsed
	}
	if c.LintIssues == "" || c.LintIssues == "[]" {
		c.lintIssuesParsed = []map[string]any{}
		return c.lintIssuesParsed
	}

	var issues []map[string]any
	err := json.Unmarshal([]byte(c.LintIssues), &issues)
	if err != nil {
		slog.Error("Error parsing lint issues", "error", err)
		return []map[string]any{}
	}
	c.lintIssuesParsed = issues

	return issues
}

func (c *Command) GetID() resource.ID {
	return c.ID
}

func (c *Command) GetSingleLineDescription(maxChars int) string {
	if c.Title == "" {
		if len(c.Script) > maxChars {
			return c.Script[:maxChars] + "..."
		}
		return c.Script
	}
	if len(c.Title) > maxChars {
		return c.Title[:maxChars] + "..."
	}
	return c.Title
}
