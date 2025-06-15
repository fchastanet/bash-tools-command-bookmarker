package command

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/keys"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/structure"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/styles"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/top/tabs"

	"github.com/fchastanet/shell-command-bookmarker/internal/services"
	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	pkgTabs "github.com/fchastanet/shell-command-bookmarker/pkg/components/tabs"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/sort"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/filters"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

type ListMaker struct {
	App                     services.AppServiceInterface
	TableCustomActionKeyMap *keys.TableCustomActionKeyMap
	FilterKeyMap            *pkgTabs.FilterKeyMap
	NavigationKeyMap        *table.Navigation
	SortKeyMap              *sort.KeyMap
	ActionKeyMap            *table.Action
	EditorsCache            table.EditorsCacheInterface
	Styles                  *styles.Styles
	Spinner                 *spinner.Model
}

const (
	idColumnPercentWidth         = 6
	titleColumnPercentWidth      = 19
	scriptColumnPercentWidth     = 65
	statusColumnPercentWidth     = 7
	lintStatusColumnPercentWidth = 6

	indexColumnStatus = 3

	percent    = 100
	sidesCount = 2
)

func (mm *ListMaker) Make(_ resource.ID, width, height int) (structure.ChildModel, error) {
	idColumn := newColumn(table.ColumnKey(structure.FieldID), "Id", table.NoTruncate)
	titleColumn := newColumn(table.ColumnKey(structure.FieldTitle), "Title", table.GetDefaultTruncationFunc())
	scriptColumn := newColumn(table.ColumnKey(structure.FieldScript), "Script", table.GetDefaultTruncationFunc())
	statusColumn := newColumn(table.ColumnKey(structure.FieldStatus), "Status", table.GetDefaultTruncationFunc())
	lintStatusColumn := newColumn(table.ColumnKey(structure.FieldLintStatus), "Lint", table.GetDefaultTruncationFunc())
	filterScoreColumn := newColumn(table.ColumnKey(structure.FieldFilterScore), "Score", table.NoTruncate)

	// set filter
	filter := filters.NewInput()
	// Initialize the category tabs component
	categoryAdapter := tabs.NewCategoryAdapter(
		mm.App.GetHistoryService(),
		mm.Styles.SortStyles,
		mm.SortKeyMap,
	)
	categoryTabs := pkgTabs.NewCategoryTabs(
		mm.Styles.CategoryTabStyles,
		filter,
		categoryAdapter,
		mm.FilterKeyMap,
		compareBySortField,
	)

	m := &commandsList{
		AppService:              mm.App.Self(),
		Model:                   nil,
		editorsCache:            mm.EditorsCache,
		tableCustomActionKeyMap: mm.TableCustomActionKeyMap,
		reloading:               false,
		spinner:                 mm.Spinner,
		width:                   width,
		height:                  height,
		styles:                  mm.Styles,
		idColumn:                &idColumn,
		titleColumn:             &titleColumn,
		scriptColumn:            &scriptColumn,
		statusColumn:            &statusColumn,
		lintStatusColumn:        &lintStatusColumn,
		filterScoreColumn:       &filterScoreColumn,
		categoryTabs:            categoryTabs,
	}
	renderer := func(cmd *dbmodels.Command) table.RenderedRow {
		return mm.renderRow(cmd, m)
	}
	headerCellRenderer := func(cellContent string, colIndex int) string {
		// Get the sort state from the active category tab
		sortState := m.categoryTabs.GetActiveSortState()

		columns := m.getColumns()
		if colIndex < 0 || colIndex >= len(columns) {
			// If the column index is out of bounds, return the cell content as is
			return cellContent
		}
		column := columns[colIndex]

		// Apply primary sort indicator
		primaryField := table.ColumnKey(sortState.PrimarySort.Field)
		if column.Key == primaryField {
			cellContent = "1" + string(sortState.PrimarySort.Direction) + " " + cellContent
		}

		// Apply secondary sort indicator if applicable
		if sortState.SecondarySort != nil {
			secondaryField := table.ColumnKey(sortState.SecondarySort.Field)
			if column.Key == secondaryField {
				cellContent = " 2" + string(sortState.SecondarySort.Direction) + " " + cellContent
			}
		}

		return cellContent
	}

	cellRenderer := func(_ *dbmodels.Command, cellContent string, colIndex int, rowEdited bool) string {
		if rowEdited && colIndex == indexColumnStatus {
			cellContent = m.styles.TableStyle.GetTableCellEditedStyle().Render("Edited")
		}
		return cellContent
	}
	// Create a dynamic sort function that always uses the current active category tab's sort state
	customSortFunc := sort.CommandSortFuncDynamic(
		func() *sort.State[*dbmodels.Command, string] {
			return categoryTabs.GetActiveSortState()
		},
	)

	tbl := table.New(
		mm.EditorsCache,
		mm.Styles.TableStyle,
		m.getColumns(),
		renderer,
		cellRenderer,
		headerCellRenderer,
		width,
		height,
		table.WithSortFunc(customSortFunc),
		table.WithPreview[*dbmodels.Command](structure.CommandKind),
		table.WithNavigation[*dbmodels.Command](mm.NavigationKeyMap),
		table.WithAction[*dbmodels.Command](mm.ActionKeyMap),
	)

	m.Model = &tbl

	return m, nil
}

func newColumn(key table.ColumnKey, title string, truncationFunc table.TruncationFunc) table.Column {
	return table.Column{
		Key:            key,
		Title:          title,
		FlexFactor:     0,
		Width:          0,
		TruncationFunc: truncationFunc,
		RightAlign:     false,
	}
}

func (*ListMaker) renderRow(
	cmd *dbmodels.Command,
	commandsListModel *commandsList,
) table.RenderedRow {
	return table.RenderedRow{
		commandsListModel.idColumn.Key:          fmt.Sprintf("%d", cmd.GetID()),
		commandsListModel.titleColumn.Key:       cmd.Title,
		commandsListModel.scriptColumn.Key:      cmd.Script,
		commandsListModel.statusColumn.Key:      formatStatus(cmd, commandsListModel.styles.EditorStyle),
		commandsListModel.lintStatusColumn.Key:  formatLintStatus(cmd, commandsListModel.styles.EditorStyle),
		commandsListModel.filterScoreColumn.Key: strconv.Itoa(cmd.FilterScore),
	}
}

func formatStatus(
	cmd *dbmodels.Command,
	editorStyle *styles.EditorStyle,
) string {
	switch cmd.Status {
	case dbmodels.CommandStatusSaved:
		return editorStyle.StatusOK.Render(string(cmd.Status))
	case dbmodels.CommandStatusImported:
		return editorStyle.ReadonlyValue.Render(string(cmd.Status))
	case dbmodels.CommandStatusObsolete:
		return editorStyle.StatusDisabled.Render(string(cmd.Status))
	case dbmodels.CommandStatusDeleted:
		return editorStyle.StatusWarning.Render(string(cmd.Status))
	default:
		return string(cmd.Status)
	}
}

func formatLintStatus(
	cmd *dbmodels.Command,
	editorStyle *styles.EditorStyle,
) string {
	switch cmd.LintStatus {
	case dbmodels.LintStatusOK:
		return editorStyle.StatusOK.Render("OK")
	case dbmodels.LintStatusWarning:
		return editorStyle.StatusWarning.Render("Warning")
	case dbmodels.LintStatusError:
		return editorStyle.StatusError.Render("Error")
	case dbmodels.LintStatusShellcheckFailed:
		return editorStyle.StatusError.Render("Shellcheck Failed")
	case dbmodels.LintStatusNotAvailable:
		return editorStyle.StatusDisabled.Render("Not Available")
	default:
		return editorStyle.StatusDisabled.Render("Not Available")
	}
}

type commandsList struct {
	editorsCache table.EditorsCacheInterface
	Model        *table.Model[*dbmodels.Command]
	*services.AppService
	styles                  *styles.Styles
	spinner                 *spinner.Model
	tableCustomActionKeyMap *keys.TableCustomActionKeyMap
	categoryTabs            *pkgTabs.CategoryTabs[
		*dbmodels.Command,
		dbmodels.CommandStatus,
		string,
	]

	idColumn          *table.Column
	titleColumn       *table.Column
	scriptColumn      *table.Column
	statusColumn      *table.Column
	lintStatusColumn  *table.Column
	filterScoreColumn *table.Column

	height int
	width  int

	reloading bool
}

func (*commandsList) BeforeSwitchPane() tea.Cmd {
	return nil
}

func (m *commandsList) getColumns() []table.Column {
	columns := []table.Column{
		*m.idColumn,
		*m.titleColumn,
		*m.scriptColumn,
		*m.statusColumn,
		*m.lintStatusColumn,
	}
	if m.categoryTabs.GetActiveFilter() != "" {
		columns = append(columns, *m.filterScoreColumn)
	}
	return columns
}

func (m *commandsList) computeColumnsWidth(width int) {
	columnsCount := 5
	spaceForAdditionalColumn := 0
	if m.categoryTabs.GetActiveFilter() != "" {
		columnsCount++
		spaceForAdditionalColumn = 1
	}

	const roundedAdaptation = 1

	w := width -
		columnsCount*m.styles.TableStyle.GetTableCellStyle().GetHorizontalPadding()*sidesCount
	m.idColumn.Width = (idColumnPercentWidth-spaceForAdditionalColumn)*w/percent + roundedAdaptation
	m.titleColumn.Width = (titleColumnPercentWidth-spaceForAdditionalColumn)*w/percent + roundedAdaptation
	m.scriptColumn.Width = (scriptColumnPercentWidth-spaceForAdditionalColumn)*w/percent + roundedAdaptation
	m.statusColumn.Width = (statusColumnPercentWidth-spaceForAdditionalColumn)*w/percent + roundedAdaptation
	m.lintStatusColumn.Width = (lintStatusColumnPercentWidth-spaceForAdditionalColumn)*w/percent + roundedAdaptation
	if m.categoryTabs.GetActiveFilter() != "" {
		m.filterScoreColumn.Width = columnsCount
	} else {
		m.filterScoreColumn.Width = 0 // Hide the filter score column if no filter is active
	}
	m.Model.SetColumns(m.getColumns())
}

func (m *commandsList) Init() tea.Cmd {
	activeSortState := m.categoryTabs.GetActiveSortState()

	return tea.Batch(
		activeSortState.Init(),
		func() tea.Msg {
			return tea.FocusMsg{}
		},
	)
}

//nolint:cyclop // not really complex
func (m *commandsList) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	keyMsg := false

	switch msg := msg.(type) {
	case sort.MsgSortEditModeChanged[*dbmodels.Command, string]:
		return m.reloadCommandsAfterSort(msg.State, nil)
	case sort.Msg[*dbmodels.Command, string]:
		return m.reloadCommandsAfterSort(msg.State, msg.InfoMsg)
	case table.ReloadMsg[*dbmodels.Command]:
		return m.loadCommandsForCurrentCategory(msg.RowID)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case tea.BlurMsg:
		m.Model.Blur()
		m.categoryTabs.Blur()
	case tea.FocusMsg:
		m.categoryTabs.Focus()
		return m.handleFocus()
	case tea.KeyMsg:
		keyMsg = true
		cmd, forward := m.handleKeyMsg(msg)
		if !forward {
			return cmd
		}
		cmds = append(cmds, cmd)
	case table.RowDeleteActionMsg[*dbmodels.Command]:
		return m.handleDeleteRows()
	case pkgTabs.CategoryTabChangedMsg[
		*dbmodels.Command,
		dbmodels.CommandStatus,
		string,
	]:
		return m.loadCommandsForCurrentCategory(-1)
	}

	// First update category tabs
	catCmd := m.categoryTabs.Update(msg)
	cmds = append(cmds, catCmd)
	if keyMsg && catCmd != nil {
		return tea.Batch(cmds...)
	}
	// Then update table model
	cmd := m.Model.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (m *commandsList) reloadCommandsAfterSort(
	state *sort.State[*dbmodels.Command, string],
	infoMsg *tui.InfoMsg,
) tea.Cmd {
	// Update the sort state in the active category tab
	m.categoryTabs.SetActiveSortState(state)

	// Reload the table with the new sort function
	reload := m.loadCommandsForCurrentCategory(-1)

	// If there's an info message, return it as well
	if infoMsg != nil {
		return tea.Batch(
			reload,
			func() tea.Msg {
				return infoMsg
			},
		)
	}
	return reload
}

// loadCommandsForCurrentCategory loads commands for the current category
func (m *commandsList) loadCommandsForCurrentCategory(selectRowID resource.ID) tea.Cmd {
	return func() tea.Msg {
		// Get status types from current category
		statuses := m.categoryTabs.GetActiveTabCommandTypes()

		// Log the current category and statuses for debugging
		slog.Debug("Loading commands for category",
			"category", m.categoryTabs.GetActiveCategory(),
			"statuses", statuses)

		// Load commands for those statuses
		rows, err := m.HistoryService.GetCommandsByStatus(statuses...)
		if err != nil {
			slog.Error("Error getting commands for category", "error", err)
			return nil
		}

		// filter commands using filter
		if m.categoryTabs.GetActiveFilter() != "" {
			filteredRows := make([]*dbmodels.Command, 0, len(rows))
			for _, cmd := range rows {
				if match, score := matchFilter(m.categoryTabs.GetActiveFilter(), cmd); match {
					cmd.FilterScore = score
					filteredRows = append(filteredRows, cmd)
				}
			}
			rows = filteredRows
		}

		// Update category counts
		m.updateCategoryCounts()

		m.computeColumnsWidth(m.width)

		// Log the number of loaded commands
		slog.Debug("Loaded commands", "count", len(rows), "statuses", statuses)

		// Return bulk insert message with the filtered commands
		info := fmt.Sprintf(
			"Loaded %d command(s) for category '%s'",
			len(rows),
			m.categoryTabs.GetActiveTabTitle(),
		)
		if m.categoryTabs.GetActiveFilter() != "" {
			info += fmt.Sprintf(" (filter: %s)", m.categoryTabs.GetActiveFilter())
		}
		return table.BulkInsertMsg[*dbmodels.Command]{
			Items:       rows,
			InfoMsg:     info,
			SelectRowID: selectRowID,
		}
	}
}

// updateCategoryCounts updates the count of commands in each category
func (m *commandsList) updateCategoryCounts() {
	// Using the CategoryTabs adapter to update counts directly from the HistoryService
	if err := m.categoryTabs.UpdateCategoryCounts(); err != nil {
		slog.Error("Error updating category counts", "error", err)
	}
}

func (m *commandsList) handleWindowSize(msg tea.WindowSizeMsg) tea.Cmd {
	// Update component layouts

	m.width = msg.Width
	m.height = msg.Height
	slog.Debug("handleWindowSize command_list", "height", m.height)

	// Update category tabs with new size
	cmd := m.categoryTabs.Update(msg)

	// Reserve space for category tabs (3 lines: tabs row, separator, filter status)
	const categoryTabsHeight = 3
	const filterHeight = 1

	// Adjust table height to make room for category tabs
	tableHeight := max(m.height-categoryTabsHeight-filterHeight, 1)

	m.Model.SetHeight(tableHeight)

	// Update columns for new width
	m.computeColumnsWidth(m.width)

	return cmd
}

func (m *commandsList) handleFocus() tea.Cmd {
	// When focused, load commands for the current category
	return m.loadCommandsForCurrentCategory(-1)
}

func (m *commandsList) handleDeleteRows() tea.Cmd {
	rows := m.Model.SelectedOrCurrent()
	if len(rows) == 0 {
		return func() tea.Msg {
			return tui.ErrorMsg(&ErrNoCommandsSelected{})
		}
	}
	for _, row := range rows {
		if row.Status == dbmodels.CommandStatusDeleted {
			return func() tea.Msg {
				return tui.ErrorMsg(&ErrSelectionMismatch{})
			}
		}
	}
	if len(rows) == 1 {
		cmd := rows[0]
		const maxCmdDetailsLength = 50
		cmdDetails := cmd.GetSingleLineDescription(maxCmdDetailsLength)
		confirmMessage := fmt.Sprintf(
			"Delete command #%d: %s?",
			cmd.GetID(),
			cmdDetails,
		)
		return tui.YesNoPrompt(
			confirmMessage,
			keys.GetFormKeyMap(),
			func() tea.Cmd {
				return m.deleteOneCommand(cmd)
			},
		)
	}

	confirmMessage := fmt.Sprintf("Delete %d commands?", len(rows))
	return tui.YesNoPrompt(
		confirmMessage,
		keys.GetFormKeyMap(),
		func() tea.Cmd {
			return m.deleteCommands(rows)
		},
	)
}

func (m *commandsList) deleteOneCommand(cmd *dbmodels.Command) tea.Cmd {
	nextRowID := m.Model.GetNextRowIDRelativeToCurrentRow()
	// Mark the command as deleted in the database
	originalStatus := cmd.Status
	cmd.Status = dbmodels.CommandStatusDeleted
	err := m.DBService.UpdateCommand(cmd)
	if err != nil {
		slog.Error("Error marking command as deleted", "error", err, "id", cmd.GetID())
		// Revert status change if update fails
		cmd.Status = originalStatus
		return tui.ReportError(fmt.Errorf("failed to mark command as deleted: %w", err))
	}

	// Return a message that will trigger the reload
	infoMsg := tui.InfoMsg(fmt.Sprintf(
		"Command #%d marked as deleted", cmd.GetID(),
	))
	return tui.CmdHandler(table.ReloadMsg[*dbmodels.Command]{
		RowID:   nextRowID,
		InfoMsg: &infoMsg,
	})
}

func (m *commandsList) deleteCommands(cmds []*dbmodels.Command) tea.Cmd {
	nextRowID := m.Model.GetNextRowIDRelativeToCurrentSelection()
	for _, cmd := range cmds {
		// Mark the commands as deleted in the database
		originalStatus := cmd.Status
		cmd.Status = dbmodels.CommandStatusDeleted
		err := m.DBService.UpdateCommand(cmd)
		if err != nil {
			slog.Error("Error marking one of the commands as deleted", "error", err, "id", cmd.GetID())
			// Revert status change if update fails
			cmd.Status = originalStatus
			return tui.ReportError(fmt.Errorf("failed to mark command as deleted: %w", err))
		}
	}

	// Return a message that will trigger the reload
	infoMsg := tui.InfoMsg(fmt.Sprintf(
		"%d commands marked as deleted", len(cmds),
	))
	return tui.CmdHandler(table.ReloadMsg[*dbmodels.Command]{
		RowID:   nextRowID,
		InfoMsg: &infoMsg,
	})
}

func (m *commandsList) handleKeyMsg(msg tea.KeyMsg) (cmd tea.Cmd, forward bool) {
	if m.categoryTabs.FilterActive() {
		return nil, true
	}
	forward = true
	activeSortState := m.categoryTabs.GetActiveSortState()
	cmd, propagate := activeSortState.Update(msg)
	if !propagate {
		return cmd, false
	}
	var cmds []tea.Cmd
	cmds = append(cmds, cmd)

	customK := m.tableCustomActionKeyMap
	switch {
	case tui.CheckKey(msg, customK.ComposeCommand):
		forward = false
		cmds = append(cmds, m.handleComposeCommand())
	case tui.CheckKey(msg, customK.RestoreCommand):
		forward = false
		cmds = append(cmds, m.handleRestoreCommand())
	case tui.CheckKey(msg, customK.CopyToClipboard):
		forward = false
		cmds = append(cmds, m.handleCopyToClipboard())
	case tui.CheckKey(msg, customK.SelectForShell):
		forward = false
		cmds = append(cmds, m.handleSelectForShell())
	}
	return tea.Batch(cmds...), forward
}

func (m *commandsList) handleRestoreCommand() tea.Cmd {
	rows := m.Model.SelectedOrCurrent()
	if len(rows) == 0 {
		return func() tea.Msg {
			return tui.ErrorMsg(&ErrNoCommandsSelected{})
		}
	}
	for _, row := range rows {
		if row.Status != dbmodels.CommandStatusDeleted {
			return func() tea.Msg {
				return tui.ErrorMsg(&ErrSelectionMismatch{})
			}
		}
	}
	err := m.HistoryService.RestoreCommand(rows)
	if err != nil {
		return func() tea.Msg {
			return tui.ErrorMsg(&ErrRestoreCommand{Err: err})
		}
	}
	m.Model.DeselectAll()

	infoMsg := tui.InfoMsg(fmt.Sprintf("Restored %d command(s)", len(rows)))
	return func() tea.Msg {
		return table.ReloadMsg[*dbmodels.Command]{
			RowID:   rows[0].GetID(),
			InfoMsg: &infoMsg,
		}
	}
}

func (m *commandsList) handleComposeCommand() tea.Cmd {
	rows := m.Model.SelectedOrCurrent()
	newCmd, err := m.HistoryService.ComposeCommand(rows)
	if err != nil {
		return func() tea.Msg {
			return tui.ErrorMsg(&ErrComposeCommand{Err: err})
		}
	}
	m.Model.DeselectAll()
	infoMsg := tui.InfoMsg(fmt.Sprintf(
		"New Command #%d created from %d selected commands", newCmd.GetID(), len(rows),
	))
	// change the category tab to "Available Commands" immediately
	// so the user can see the new command right away
	// The sort state will be automatically loaded from the target category tab
	m.categoryTabs.ChangeCategoryTab(tabs.AvailableCommands)
	return tea.Batch(
		func() tea.Msg {
			return table.ReloadMsg[*dbmodels.Command]{
				RowID:   newCmd.ID,
				InfoMsg: &infoMsg,
			}
		},
	)
}

func (m *commandsList) handleCopyToClipboard() tea.Cmd {
	rows := m.Model.SelectedOrCurrent()
	if len(rows) == 0 {
		return func() tea.Msg {
			return tui.ErrorMsg(&ErrNoCommandsSelected{})
		}
	}

	commandsString := m.HistoryService.CreateCommandsString(rows)
	err := clipboard.WriteAll(commandsString)
	if err != nil {
		return func() tea.Msg {
			return tui.ErrorMsg(&ErrClipboardCopyFailed{Err: err})
		}
	}

	m.Model.DeselectAll()
	return func() tea.Msg {
		return tui.InfoMsg(fmt.Sprintf("Copied %d command(s) to clipboard", len(rows)))
	}
}

func (m *commandsList) handleSelectForShell() tea.Cmd {
	rows := m.Model.SelectedOrCurrent()
	if len(rows) == 0 {
		return func() tea.Msg {
			return tui.ErrorMsg(&ErrNoCommandsSelected{})
		}
	}

	// We only want the first command for shell pasting
	commandString := m.HistoryService.CreateCommandsString(rows[:1])

	return func() tea.Msg {
		return structure.CommandSelectedForShellMsg{Command: commandString}
	}
}

func (m *commandsList) View() string {
	if m.reloading {
		return "Pulling state " + m.spinner.View()
	}

	// Make sure we have valid dimensions
	if m.width <= 0 || m.height <= 0 {
		return "Waiting for proper window dimensions..."
	}

	// Render category tabs
	categoryTabsView := m.categoryTabs.View()

	// Add a horizontal line as separator
	separator := strings.Repeat("-", m.width)

	// Display table view
	tableView := m.Model.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		categoryTabsView,
		separator,
		tableView,
	)
}
