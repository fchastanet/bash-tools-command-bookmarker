package keys

import (
	"github.com/charmbracelet/bubbles/key"
	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

type TableCustomActionKeyMap struct {
	ComposeCommand  *key.Binding
	CopyToClipboard *key.Binding
	SelectForShell  *key.Binding
	RestoreCommand  *key.Binding
}

func GetTableCustomActionKeyMap() *TableCustomActionKeyMap {
	composeCommand := key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "compose command"),
	)
	copyToClipboard := key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "copy to clipboard"),
	)
	selectForShell := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select for shell"),
	)
	restoreCommand := key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restore command"),
	)

	return &TableCustomActionKeyMap{
		ComposeCommand:  &composeCommand,
		CopyToClipboard: &copyToClipboard,
		SelectForShell:  &selectForShell,
		RestoreCommand:  &restoreCommand,
	}
}

// Navigation returns key bindings for table.
func GetTableNavigationKeyMap() *table.Navigation {
	return table.GetDefaultNavigation()
}

func GetTableActionKeyMap() *table.Action {
	return table.GetDefaultAction()
}

func UpdateBindings(
	tableActions *table.Action,
	tableCustomActions *TableCustomActionKeyMap,
	shellSelectionMode bool,
	selectedCommand *dbmodels.Command,
) {
	tableCustomActions.SelectForShell.SetEnabled(
		selectedCommand != nil &&
			shellSelectionMode &&
			selectedCommand.Status != dbmodels.CommandStatusDeleted,
	)
	tableCustomActions.ComposeCommand.SetEnabled(
		!shellSelectionMode && selectedCommand != nil && selectedCommand.IsEditable(),
	)
	tableCustomActions.CopyToClipboard.SetEnabled(
		selectedCommand != nil,
	)
	tableCustomActions.RestoreCommand.SetEnabled(
		!shellSelectionMode &&
			selectedCommand != nil &&
			selectedCommand.Status == dbmodels.CommandStatusDeleted,
	)
	tableActions.Delete.SetEnabled(
		!shellSelectionMode &&
			selectedCommand != nil &&
			selectedCommand.Status != dbmodels.CommandStatusDeleted,
	)
	tableActions.Select.SetEnabled(selectedCommand != nil)
	tableActions.SelectAll.SetEnabled(selectedCommand != nil)
	tableActions.SelectClear.SetEnabled(selectedCommand != nil)
	tableActions.SelectRange.SetEnabled(selectedCommand != nil)
	tableActions.Enter.SetEnabled(selectedCommand != nil && !shellSelectionMode)
}
