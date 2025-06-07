package services

import (
	"github.com/fchastanet/shell-command-bookmarker/internal/args"
	"github.com/fchastanet/shell-command-bookmarker/internal/services/models"
)

type CommandExecutorInterface interface {
	// ExecuteCommandWithStdin executes a command with stdin and returns the output and error if any.
	ExecuteCommandWithStdin(cmd string, args []string, stdin string) (
		output string, errOutput string, err error,
	)
}

type LookupExecutorInterface interface {
	LookPath(path string) (string, error)
}

// AppServiceInterface defines the expected behavior of an AppService
type AppServiceInterface interface {
	Main(cli *args.Cli, sqliteSchema string) error
	IsTerminalCompatible() error
	IsShellSelectionMode() bool
	Init(cfg AppServiceConfig) error
	Cleanup()
	GetHistoryService() *HistoryService
	HandleShellIntegrationScriptGeneration(cli *args.Cli) bool
	Self() *AppService
}

// HistoryServiceInterface defines the expected behavior of a HistoryService
type HistoryServiceInterface interface {
	GetCommandsByStatus(statuses ...models.CommandStatus) ([]*models.Command, error)
	GetCommandCountsByStatus() (map[models.CommandStatus]int, error)
	GetCommandCountsByCategory() (map[CommandCategory]int, error)
	GetCommandStatusesByCategory(category CommandCategory) []models.CommandStatus
	GetCommandCategoryTitles() map[CommandCategory]string
	GetAllCommandCategories() []CommandCategory
	IngestHistory() error
	UpdateCommand(command *models.Command) (*models.Command, error)
	ComposeCommand(commands []*models.Command) (*models.Command, error)
	CreateCommandsString(commands []*models.Command) string
}
