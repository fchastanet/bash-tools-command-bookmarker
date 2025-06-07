package services

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fchastanet/shell-command-bookmarker/internal/processors"
	"github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
)

const (
	// MinCommandLength is the minimum length of a command to be ingested
	MinCommandLength = 6
)

type HistoryIngestor interface {
	// IngestHistoryWithCallback reads the bash history file and ingests it into the database using a callback
	ParseBashHistory(
		historyFilePath string, fromTimestamp time.Time,
		callback func(processors.HistoryCommand) (processors.CommandImportedStatus, error),
	) error
}

// CommandCategory defines the categories of commands shown in the UI
type CommandCategory string

const (
	// CommandCategoryAvailable represents commands that are available for use
	CommandCategoryAvailable CommandCategory = "available"
	// CommandCategoryBookmarked represents commands that have been saved
	CommandCategorySaved CommandCategory = "saved"
	// CommandCategoryNew represents commands that have been imported but not yet saved
	CommandCategoryNew CommandCategory = "new"
	// CommandCategoryDeleted represents commands that have been marked as deleted
	CommandCategoryDeleted CommandCategory = "deleted"
	// CommandCategoryAll represents all commands regardless of status
	CommandCategoryAll CommandCategory = "all"
)

type HistoryService struct {
	ingestor          HistoryIngestor
	homeDir           string
	dbService         *DBService
	lintService       *LintService
	scriptRegexp      *regexp.Regexp
	ignoreLinesRegexp []*regexp.Regexp
}

func NewHistoryService(
	ingestor HistoryIngestor,
	dbService *DBService,
	lintService *LintService,
) *HistoryService {
	return &HistoryService{
		ingestor:          ingestor,
		dbService:         dbService,
		lintService:       lintService,
		homeDir:           "",
		scriptRegexp:      nil,
		ignoreLinesRegexp: nil,
	}
}

// GetCommandsByStatus returns commands filtered by specific status types
func (s *HistoryService) GetCommandsByStatus(statuses ...models.CommandStatus) ([]*models.Command, error) {
	cmds, err := s.dbService.GetCommands(statuses...)
	if err != nil {
		slog.Error("Error getting commands by status", "statuses", statuses, "error", err)
		return []*models.Command{}, err
	}

	return cmds, nil
}

// GetCommandCountsByStatus returns a map of counts for each command status
func (s *HistoryService) GetCommandCountsByStatus() (map[models.CommandStatus]int, error) {
	counts, err := s.dbService.GetCommandCountsByStatus()
	if err != nil {
		slog.Error("Error getting command counts by status", "error", err)
		return nil, err
	}

	return counts, nil
}

func (s *HistoryService) getScriptRegexp() *regexp.Regexp {
	if s.scriptRegexp != nil {
		return s.scriptRegexp
	}
	s.scriptRegexp = regexp.MustCompile("[|&;><()\\[\\]{}$*?!+=,`]")
	return s.scriptRegexp
}

func (s *HistoryService) getIgnoreLinesRegexp() []*regexp.Regexp {
	if s.ignoreLinesRegexp != nil {
		return s.ignoreLinesRegexp
	}

	s.ignoreLinesRegexp = []*regexp.Regexp{
		regexp.MustCompile("^#"),
		regexp.MustCompile("( --version| --help)"),
		regexp.MustCompile("^(shutdown|export|kill|ln|man|mc|ls|ll|ps|source|which|command -v|cd|pwd|echo|cat|rm|mv|cp|touch|mkdir|rmdir|chmod|chown|top|killall|grep|find|locate|updatedb|z) "),
		regexp.MustCompile("^(code|vi|vim|nano|exit|logout|clear|history|alias|unalias|export|unset|set|env|source|bash|sh|zsh) "),
		regexp.MustCompile(`^(\./[^ ]+|exit|ls|alias|cd)$`),
		regexp.MustCompile(`^[A-Za-z0-9_]+=[^ ]+$`),
		regexp.MustCompile(`^\s*$`),
	}
	return s.ignoreLinesRegexp
}

func (s *HistoryService) Init() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Error getting home directory", "error", err)
		return err
	}
	s.homeDir = homeDir
	return nil
}

func (s *HistoryService) getDefaultHistoryFilePath() (string, error) {
	historyFile := filepath.Join(s.homeDir, ".bash_history")
	if _, err := os.Stat(historyFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Error("History file does not exist", "file", historyFile, "error", err)
			return "", err
		}
		if errors.Is(err, os.ErrPermission) {
			slog.Error("Permission denied to access history file", "file", historyFile, "error", err)
			return "", err
		}
	}
	return historyFile, nil
}

func (s *HistoryService) getHistoryFilePath() (string, error) {
	historyFile := os.Getenv("HISTFILE")
	if historyFile == "" {
		historyFile, err := s.getDefaultHistoryFilePath()
		if err != nil {
			slog.Error("Error getting default history file path", "error", err)
			return "", err
		}
		slog.Info("Using default history file path", "file", historyFile)
		return historyFile, nil
	}
	_, err := os.Stat(historyFile)
	if errors.Is(err, os.ErrNotExist) {
		slog.Error("History file does not exist", "file", historyFile, "error", err)
		return "", err
	}
	if errors.Is(err, os.ErrPermission) {
		slog.Error("Permission denied to access history file", "file", historyFile, "error", err)
		return "", err
	}
	return historyFile, nil
}

func (s *HistoryService) checkIfCommandShouldBeSaved(cmd processors.HistoryCommand) (processors.CommandImportedStatus, error) {
	if len(cmd.Command) < MinCommandLength {
		slog.Info("Command too short, skipping", "command", cmd)
		return processors.CommandImportedStatusSkipped, nil
	}
	if !s.getScriptRegexp().MatchString(cmd.Command) &&
		matchOneOfRegexp(cmd.Command, s.getIgnoreLinesRegexp()) {
		slog.Info("Command does not match any script or is ignored", "command", cmd)
		return processors.CommandImportedStatusFilteredOut, nil
	}

	existingCmd, err := s.dbService.GetCommandByScript(cmd.Command)
	if err != nil {
		slog.Error("Error getting command from database", "command", cmd, "error", err)
		return processors.CommandImportedStatusError, err
	}
	if existingCmd != nil {
		slog.Debug("Command already exists in database", "command", cmd)
		return processors.CommandImportedStatusAlreadyExists, nil
	}
	return processors.CommandImportedStatusNew, nil
}

func (s *HistoryService) IngestHistory() error {
	historyFilePath, err := s.getHistoryFilePath()
	if err != nil {
		slog.Error("Error getting history file path", "error", err)
		return err
	}

	if historyFilePath == "" {
		slog.Warn("No history file path provided")
		return nil
	}

	maxCommandTimestamp, err := s.dbService.GetMaxCommandTimestamp()
	if err != nil {
		slog.Debug("Error getting max command timestamp, fallback to 0", "error", err)
		maxCommandTimestamp = time.Time{}
	}
	slog.Debug("Max command timestamp", "timestamp", maxCommandTimestamp)

	if err := s.ingestor.ParseBashHistory(historyFilePath, maxCommandTimestamp, s.processCmd); err != nil {
		slog.Error("Error ingesting history", "file", historyFilePath, "error", err)
		return err
	}

	return nil
}

func (s *HistoryService) processCmd(historyCmd processors.HistoryCommand) (processors.CommandImportedStatus, error) {
	if importStatus, err := s.checkIfCommandShouldBeSaved(historyCmd); err != nil {
		return processors.CommandImportedStatusError, err
	} else if importStatus != processors.CommandImportedStatusNew {
		slog.Debug("Command already exists in database or is ignored", "command", historyCmd, "status", importStatus)
		return importStatus, nil
	}
	cmd := models.NewCommand(
		historyCmd.Command,
		historyCmd.Elapsed,
		historyCmd.Timestamp,
	)

	s.lintService.LintCommand(cmd)
	if err := s.dbService.SaveCommand(cmd); err != nil {
		slog.Error("Error saving command to database", "command", cmd, "error", err)
		return processors.CommandImportedStatusError, err
	}
	slog.Info("Command saved successfully", "command", cmd)
	return processors.CommandImportedStatusNew, nil
}

func (s *HistoryService) UpdateCommand(command *models.Command) (newCommand *models.Command, err error) {
	slog.Debug("Updating command", "id", command.ID, "status", command.Status)

	// If it's an IMPORTED command being updated, we need to handle duplication
	if command.Status == models.CommandStatusImported {
		if err := s.duplicateCommandAsObsolete(command.ID); err != nil {
			slog.Error("Failed to duplicate original command as obsolete", "id", command.ID, "error", err)
			return nil, err
		}
	}

	// For non-IMPORTED commands, just update directly
	command.ModificationDatetime = time.Now()
	command.Status = models.CommandStatusSaved
	// Lint the new command
	s.lintCommand(command)
	err = s.dbService.UpdateCommand(command)
	if err != nil {
		slog.Error("Error updating command in database", "id", command.ID, "error", err)
		return nil, err
	}
	return command, nil
}

func (s *HistoryService) lintCommand(command *models.Command) {
	if command.Status != models.CommandStatusSaved {
		slog.Warn("Command is not in a state that can be linted", "id", command.ID, "status", command.Status)
		return
	}

	s.lintService.LintCommand(command)
}

func (s *HistoryService) duplicateCommandAsObsolete(commandID resource.ID) error {
	// Save the new command
	newID, err := s.dbService.DuplicateCommand(commandID, models.CommandStatusObsolete)
	if err != nil {
		slog.Error("Error duplicating command as obsolete", "error", err)
		return err
	}
	slog.Info("Original command duplicated as obsolete", "id", newID, "originalID", commandID)
	return nil
}

func matchOneOfRegexp(line string, regexps []*regexp.Regexp) bool {
	for _, r := range regexps {
		if r.MatchString(line) {
			return true
		}
	}
	return false
}

func (s *HistoryService) RestoreCommand(commands []*models.Command) error {
	if len(commands) < 1 {
		return &ComposeInsufficientCommandsProvidedError{nil}
	}
	for _, cmd := range commands {
		cmd.Status = models.CommandStatusSaved
		cmd.ModificationDatetime = time.Now()
		s.lintService.LintCommand(cmd)
		err := s.dbService.UpdateCommand(cmd)
		if err != nil {
			slog.Error("Error restoring command", "id", cmd.ID, "error", err)
			return err
		}
	}
	return nil
}

func (s *HistoryService) ComposeCommand(commands []*models.Command) (*models.Command, error) {
	if len(commands) < 1 {
		return nil, &ComposeInsufficientCommandsProvidedError{nil}
	}
	newCommand := models.NewCommand(
		s.generateComposeCommandScript(commands),
		0,
		time.Now(),
	)
	s.lintService.LintCommand(newCommand)

	err := s.dbService.SaveCommand(newCommand)
	return newCommand, err
}

func (s *HistoryService) generateComposeCommandScript(commands []*models.Command) string {
	var script strings.Builder
	script.WriteString("#!/usr/bin/env bash\n")
	script.WriteString("set -e -o pipefail -o errexit\n")
	for _, cmd := range commands {
		script.WriteString(fmt.Sprintf("echo '%s'\n", cmd.Script))
	}
	return script.String()
}

// CreateCommandsString creates a concatenated string of all commands
// suitable for copying to clipboard
func (s *HistoryService) CreateCommandsString(commands []*models.Command) string {
	var script strings.Builder
	for i, cmd := range commands {
		script.WriteString(cmd.Script)
		if i < len(commands)-1 {
			script.WriteString("\n")
		}
	}
	return script.String()
}

// GetCommandCountsByCategory returns a map of command counts by category
func (s *HistoryService) GetCommandCountsByCategory() (map[CommandCategory]int, error) {
	// Get the raw counts by status
	statusCounts, err := s.dbService.GetCommandCountsByStatus()
	if err != nil {
		slog.Error("Error getting command counts by status", "error", err)
		return nil, err
	}

	// Map database status counts to category counts
	categoryCounts := make(map[CommandCategory]int)

	// Available commands (showing Imported status)
	categoryCounts[CommandCategoryAvailable] = statusCounts[models.CommandStatusImported] +
		statusCounts[models.CommandStatusSaved]

	// Saved commands (showing Saved status)
	categoryCounts[CommandCategorySaved] = statusCounts[models.CommandStatusSaved]

	// New commands (showing Imported status)
	categoryCounts[CommandCategoryNew] = statusCounts[models.CommandStatusImported]

	// Deleted commands
	categoryCounts[CommandCategoryDeleted] = statusCounts[models.CommandStatusDeleted]

	// All commands (sum of all statuses)
	categoryCounts[CommandCategoryAll] = 0
	for _, count := range statusCounts {
		categoryCounts[CommandCategoryAll] += count
	}

	return categoryCounts, nil
}
