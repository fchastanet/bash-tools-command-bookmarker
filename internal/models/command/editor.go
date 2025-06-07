// Package editor provides command editing functionality
package command

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/command/inputs"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/keys"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/structure"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/styles"
	"github.com/fchastanet/shell-command-bookmarker/internal/services"
	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
	"github.com/fchastanet/shell-command-bookmarker/pkg/utils"
)

// Error definitions
var (
	// ErrCommandNotFound is returned when a command with the given ID is not found
	ErrCommandNotFound = errors.New("command not found with ID")
)

type EditorMaker struct {
	App           *services.AppService
	Styles        *styles.Styles
	EditorKeyMap  *keys.EditorKeyMap
	commandEditor *commandEditor
}

// Number of input fields
const (
	numInputFields           = 3    // Title, Description, Script
	titleInputMaxSize        = 50   // Max size for title input
	descriptionInputMaxSize  = 1000 // Max size for description input
	descriptionInputHeight   = 5    // Height for description input
	scriptInputHeight        = 5    // Height for script input
	descriptionWordwrapWidth = 80   // Word wrap width for description input
	inputFieldPadding        = 2
)

// Make creates a new command editor model based on the command ID
func (mm *EditorMaker) Make(id resource.ID, width, height int) (structure.ChildModel, error) {
	// Create the editor model
	if mm.commandEditor == nil {
		mm.commandEditor = &commandEditor{
			AppService:    mm.App.Self(),
			styles:        mm.Styles,
			command:       nil,
			width:         width,
			height:        height,
			inputs:        make([]inputs.Input, numInputFields),
			focused:       -1,
			EditorKeyMap:  mm.EditorKeyMap,
			pagePosition:  0,
			contentHeight: 0,
			initialized:   false,
		}
		mm.commandEditor.Init()
	}
	command, err := mm.commandEditor.getCommand(id)
	if err != nil {
		return nil, err
	}
	mm.commandEditor.setCommand(command)

	return mm.commandEditor, nil
}

// commandEditor is the model for editing a command
type commandEditor struct {
	*services.AppService
	styles        *styles.Styles
	command       *dbmodels.Command
	EditorKeyMap  *keys.EditorKeyMap
	inputs        []inputs.Input
	width         int
	height        int
	focused       int
	pagePosition  int
	contentHeight int
	initialized   bool
}

func (m *commandEditor) BeforeSwitchPane() tea.Cmd {
	return m.confirmAbandonChanges(false)
}

func (m *commandEditor) getCommand(commandID resource.ID) (*dbmodels.Command, error) {
	// Load the command from the database
	command, err := m.DBService.GetCommandByID(commandID)
	if err != nil {
		return nil, fmt.Errorf("failed to load command %d: %w", commandID, err)
	}

	if command == nil {
		return nil, fmt.Errorf("%w %d", ErrCommandNotFound, commandID)
	}
	return command, nil
}

func (m *commandEditor) setCommand(command *dbmodels.Command) {
	if command == nil {
		slog.Error("setCommand called with nil command")
		return
	}
	m.command = command
	m.inputs[0].SetValue(m.command.Title)
	m.inputs[1].SetValue(m.command.Description)
	m.inputs[2].SetValue(m.command.Script)
	m.initInputs()
}

// Init initializes the command editor
func (m *commandEditor) Init() tea.Cmd {
	if m.initialized {
		return nil
	}
	// Initialize the text inputs
	titleInput := inputs.NewInputWrapper("Enter title")
	titleInput.SetCharLimit(titleInputMaxSize)
	titleInput.Focus()

	descriptionInput := inputs.NewTextAreaWrapper(
		descriptionInputHeight,
		"Enter description (markdown)",
		inputs.WithMarkdown(descriptionWordwrapWidth))
	descriptionInput.SetCharLimit(descriptionInputMaxSize)

	scriptInput := inputs.NewTextAreaWrapper(scriptInputHeight, "Enter script")

	m.inputs = []inputs.Input{titleInput, descriptionInput, scriptInput}
	m.focused = -1
	m.initialized = true

	return nil
}

// formatLintStatus returns a styled string representing the lint status
func (m *commandEditor) formatLintStatus() string {
	switch m.command.LintStatus {
	case dbmodels.LintStatusOK:
		return m.styles.EditorStyle.StatusOK.Render("OK")
	case dbmodels.LintStatusWarning:
		return m.styles.EditorStyle.StatusWarning.Render("Warning")
	case dbmodels.LintStatusError:
		return m.styles.EditorStyle.StatusError.Render("Error")
	case dbmodels.LintStatusShellcheckFailed:
		return m.styles.EditorStyle.StatusError.Render("Shellcheck Failed")
	case dbmodels.LintStatusNotAvailable:
		return m.styles.EditorStyle.StatusDisabled.Render("Not Available")
	default:
		return m.styles.EditorStyle.StatusDisabled.Render("Not Available")
	}
}

// Update handles updates to the command editor
func (m *commandEditor) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	slog.Debug("commandEditor.Update", "msgType", fmt.Sprintf("%T", msg), "focused", m.focused)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSizeMsg(msg)
	case structure.FocusedPaneChangedMsg:
		cmds = append(cmds, m.handleFocusedPaneChangedMsg(msg))
	case structure.NavigationMsg:
		cmds = append(cmds, m.handleNavigationMsg(msg))
	case table.RowSelectedActionMsg[*dbmodels.Command]:
		// This message is sent when a row is selected in the command table
		m.setCommand(msg.Row)
	case tea.KeyMsg:
		cmd := m.handleKeyMsg(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
			return tea.Batch(cmds...)
		}
		if msg.Alt {
			// avoid alt key combinations to be eaten by inputs
			return nil
		}
	}

	// Update the active input field
	if m.focused >= 0 {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m *commandEditor) handleNavigationMsg(msg structure.NavigationMsg) tea.Cmd {
	cmd, err := m.getCommand(msg.Page.ID)
	if err != nil {
		slog.Error("Failed to load command for navigation", "id", msg.Page.ID, "error", err)
		return tui.ReportError(&ErrCommandLoadingFailure{
			CommandID: msg.Page.ID,
			Err:       err,
		})
	}
	if cmd != nil {
		m.setCommand(cmd)
	} else {
		slog.Error("Failed to load command for navigation", "id", msg.Page.ID)
		return tui.ReportError(&ErrCommandLoadingFailure{
			CommandID: msg.Page.ID,
			Err:       nil,
		})
	}
	return nil
}

func (m *commandEditor) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	for _, input := range m.inputs {
		input.SetWidth(m.width - inputFieldPadding)
	}
}

func (m *commandEditor) handleFocusedPaneChangedMsg(msg structure.FocusedPaneChangedMsg) tea.Cmd {
	if m.focused >= 0 && m.inputs[m.focused] != nil {
		m.inputs[m.focused].Blur()
	}
	if msg.To == structure.BottomPane {
		m.focused = -1
	}
	m.initInputs()
	return tui.GetDummyCmd()
}

func (m *commandEditor) initInputs() {
	for i, input := range m.inputs {
		input.SetReadOnly(m.focused != i || !m.command.IsEditable())
	}
}

//nolint:cyclop // not really complex
func (m *commandEditor) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	var cmds []tea.Cmd
	editorK := m.EditorKeyMap
	switch {
	case key.Matches(msg, *editorK.PreviousField) && editorK.PreviousField.Enabled():
		cmds = append(cmds, m.prevField())
	case key.Matches(msg, *editorK.NextField) && editorK.NextField.Enabled():
		cmds = append(cmds, m.nextField())
	case key.Matches(msg, *editorK.PreviousPage) && editorK.PreviousPage.Enabled():
		m.prevPage()
	case key.Matches(msg, *editorK.NextPage) && editorK.NextPage.Enabled():
		m.nextPage()
	case key.Matches(msg, *editorK.Save) && editorK.Save.Enabled():
		return m.save()
	case key.Matches(msg, *editorK.Cancel) && editorK.Cancel.Enabled():
		return m.confirmAbandonChanges(true)
	}

	return tea.Batch(cmds...)
}

// View renders the command editor
func (m *commandEditor) View() string {
	slog.Debug("commandEditor.View", "focused", m.focused)
	var content strings.Builder

	// Add common elements first
	m.addCommonElements(&content)

	// Add readonly information section
	m.addReadonlySection(&content)

	// strip the first lines of content to fit the screen
	contentStr := content.String()
	visibleContent := utils.RemoveFirstLines(contentStr, m.pagePosition)

	// Create the content area with scrollbar
	contentArea := lipgloss.NewStyle().
		Width(m.width - m.styles.ScrollbarStyle.Width).
		Render(visibleContent + "\n")

	// Generate scrollbar
	scrollbar := m.generateScrollbar(contentStr, visibleContent)

	contentScroll := lipgloss.NewStyle().
		Height(m.height).
		MaxHeight(m.height).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, contentArea, scrollbar))

	return contentScroll
}

// addCommonElements adds the title, help text, and input fields to the content
func (m *commandEditor) addCommonElements(content *strings.Builder) {
	// Add help text at the top
	helpTextStyle := *m.styles.EditorStyle.HelpText
	if m.focused == -1 {
		helpTextStyle = helpTextStyle.Bold(true).Foreground(lipgloss.Color("255"))
	}
	var helpText string
	if m.command.IsEditable() {
		helpText = helpTextStyle.Render("⭾/Shift-⭾: Fields • ⇞/⇟: Scroll • Ctrl+S: Save • Esc: Cancel")
	} else {
		helpText = m.styles.EditorStyle.StatusWarning.Render("Command is read-only") +
			"         " + helpTextStyle.Render("⭾/Shift-⭾: Fields • ⇞/⇟: Scroll • Esc: Close")
	}
	content.WriteString(helpText + "\n\n")

	// Labels for our fields
	labels := []string{"Title:", "Description(markdown):", "Script:"}

	// Render each field with its label
	for i, label := range labels {
		labelStyle := m.styles.EditorStyle.Label
		if m.focused == i {
			labelStyle = m.styles.EditorStyle.LabelFocused
		}
		// Style the label
		styledLabel := labelStyle.Render(label)

		// Render the input field
		fmt.Fprintf(content, "%s\n%s\n\n", styledLabel, m.inputs[i].View())
	}
}

// addReadonlySection adds the readonly information section to the content
func (m *commandEditor) addReadonlySection(content *strings.Builder) {
	// Add readonly information section
	readonlyTitle := m.styles.EditorStyle.Label.Render("Readonly Information:")
	content.WriteString(readonlyTitle + "\n\n")

	// Format and add each readonly field with label and value
	createLabel := m.styles.EditorStyle.ReadonlyLabel.Render("Created:")
	createValue := m.styles.EditorStyle.ReadonlyValue.Render(
		m.command.CreationDatetime.Format(time.DateTime))

	modifyLabel := m.styles.EditorStyle.ReadonlyLabel.Render("Modified:")
	modifyValue := m.styles.EditorStyle.ReadonlyValue.Render(
		m.command.ModificationDatetime.Format(time.DateTime))

	lintStatusLabel := m.styles.EditorStyle.ReadonlyLabel.Render("Lint Status:")
	lintIssuesLabel := m.styles.EditorStyle.ReadonlyLabel.Render("Lint Issues:")

	// Add the formatted readonly information
	fmt.Fprintf(content, "%s %s\n", createLabel, createValue)
	fmt.Fprintf(content, "%s %s\n", modifyLabel, modifyValue)
	fmt.Fprintf(content, "%s %s\n", lintStatusLabel, m.formatLintStatus())

	m.addLintIssues(content, lintIssuesLabel)
}

// addLintIssues adds the lint issues section to the content
func (m *commandEditor) addLintIssues(content *strings.Builder, lintIssuesLabel string) {
	// Parse and display lint issues
	issues := m.command.GetLintIssues()
	if len(issues) == 0 {
		fmt.Fprintf(content, "%s %s\n\n", lintIssuesLabel,
			m.styles.EditorStyle.ReadonlyValue.Render("None"))
		return
	}

	fmt.Fprintf(content, "%s %s\n", lintIssuesLabel,
		m.styles.EditorStyle.ReadonlyValue.Render(fmt.Sprintf("%d issues found:", len(issues))))

	// Display each lint issue
	for i, issue := range issues {
		// Format the issue number
		num := m.styles.EditorStyle.ReadonlyLabel.Render(fmt.Sprintf("%d.", i+1))

		// Extract and format the message
		message := "Unknown issue"
		if msg, ok := issue["message"].(string); ok {
			message = msg
		}

		// Extract and format the level
		level := "unknown"
		if lvl, ok := issue["level"].(string); ok {
			level = lvl
		}

		// Style based on level
		styledMessage := m.getStyledMessage(level, message)
		fmt.Fprintf(content, "   %s %s %s\n", num, level, styledMessage)
	}
	content.WriteString("\n")
}

// getStyledMessage returns styled message based on issue level
func (m *commandEditor) getStyledMessage(level, message string) string {
	switch level {
	case "error":
		return m.styles.EditorStyle.StatusError.Render(message)
	case "warning":
		return m.styles.EditorStyle.StatusWarning.Render(message)
	case "info":
		return m.styles.EditorStyle.StatusOK.Render(message)
	default:
		return m.styles.EditorStyle.ReadonlyValue.Render(message)
	}
}

// generateScrollbar creates the scrollbar for the editor
func (m *commandEditor) generateScrollbar(contentStr, visibleContent string) string {
	// Generate scrollbar
	const minEditorHeight = 15 // Minimum height for the editor to ensure usability
	availableHeight := max(minEditorHeight, m.height)
	m.contentHeight = lipgloss.Height(contentStr)
	visibleContentHeight := lipgloss.Height(visibleContent)
	return tui.Scrollbar(
		m.styles.EditorStyle.ScrollbarStyle,
		availableHeight,
		m.contentHeight,
		min(availableHeight, visibleContentHeight),
		m.pagePosition,
	)
}

func (m *commandEditor) confirmAbandonChanges(cancel bool) tea.Cmd {
	// If no changes are made, just return
	if !m.EditionInProgress() {
		if cancel {
			return tea.Cmd(func() tea.Msg {
				return EditorCancelledMsg{}
			})
		}
		return nil
	}

	// Prompt the user for confirmation
	return tui.YesNoPrompt(
		fmt.Sprintf("Abandon changes for command #%d?", m.command.ID),
		keys.GetFormKeyMap(),
		func() tea.Cmd {
			return m.cancel()
		},
	)
}

// nextField focuses the next field
func (m *commandEditor) nextField() tea.Cmd {
	if m.focused >= 0 {
		m.inputs[m.focused].Blur()
	}
	if m.focused == len(m.inputs)-1 {
		m.focused = -1
		m.initInputs()
		return m.confirmAbandonChanges(false)
	}

	m.focused = (m.focused + 1) % len(m.inputs)
	m.initInputs()

	return m.inputs[m.focused].Focus()
}

// prevField focuses the previous field
func (m *commandEditor) prevField() tea.Cmd {
	if m.focused >= 0 {
		m.inputs[m.focused].Blur()
	}
	switch m.focused {
	case -1:
		m.focused = len(m.inputs) - 1
		m.initInputs()
		return m.confirmAbandonChanges(false)
	case 0:
		m.focused = -1
	default:
		m.focused = (m.focused - 1) % len(m.inputs)
	}
	m.initInputs()
	if m.focused >= 0 {
		return m.inputs[m.focused].Focus()
	}
	return tui.GetDummyCmd()
}

// nextPage scrolls to the next page
func (m *commandEditor) nextPage() {
	slog.Debug("nextPage", "pagePosition", m.pagePosition, "height", m.height, "contentHeight", m.contentHeight)
	m.pagePosition = min(m.pagePosition+m.height, m.contentHeight-m.height)
	slog.Debug("nextPage", "newPagePosition", m.pagePosition)
}

// prevPage scrolls to the previous page
func (m *commandEditor) prevPage() {
	slog.Debug("prevPage", "pagePosition", m.pagePosition, "height", m.height, "contentHeight", m.contentHeight)
	m.pagePosition = max(m.pagePosition-m.height, 0)
	slog.Debug("prevPage", "newPagePosition", m.pagePosition)
}

func (m *commandEditor) EditionInProgress() bool {
	return m.command.Title != m.inputs[0].Value() ||
		m.command.Description != m.inputs[1].Value() ||
		m.command.Script != m.inputs[2].Value()
}

// save saves the current command
func (m *commandEditor) save() tea.Cmd {
	// Update the command with values from the input fields
	oldTitle := m.command.Title
	oldDescription := m.command.Description
	oldScript := m.command.Script

	m.command.Title = m.inputs[0].Value()
	m.command.Description = m.inputs[1].Value()
	m.command.Script = m.inputs[2].Value()

	// Only update if there are actual changes
	if oldTitle != m.command.Title ||
		oldDescription != m.command.Description ||
		oldScript != m.command.Script {
		// Update command in database using HistoryService
		newCommand, err := m.HistoryService.UpdateCommand(m.command)
		if err != nil {
			slog.Error("Failed to save command", "id", m.command.ID, "error", err)
			return tui.ReportError(err)
		}
		m.command = newCommand

		// Trigger table reload to reflect changes
		infoMsg := tui.InfoMsg(fmt.Sprintf(
			"Command #%d saved successfully", m.command.ID,
		))

		return tui.CmdHandler(table.ReloadMsg[*dbmodels.Command]{
			RowID:   m.command.ID,
			InfoMsg: &infoMsg,
		})
	}

	return tui.ReportInfo("No changes to save for command #%d", m.command.ID)
}

type EditorCancelledMsg struct{}

// cancel returns from the editor without saving
func (m *commandEditor) cancel() tea.Cmd {
	m.revertChanges()
	infoMsg := tui.InfoMsg(fmt.Sprintf("Abandoned changes for command #%d", m.command.ID))
	return tea.Batch(
		tui.CmdHandler(EditorCancelledMsg{}),
		tui.CmdHandler(table.ReloadMsg[*dbmodels.Command]{
			RowID:   m.command.ID,
			InfoMsg: &infoMsg,
		}),
	)
}

func (m *commandEditor) revertChanges() {
	// Revert changes to the original command state
	m.inputs[0].SetValue(m.command.Title)
	m.inputs[1].SetValue(m.command.Description)
	m.inputs[2].SetValue(m.command.Script)
}

// BorderText returns text to display in the border
func (m *commandEditor) BorderText() map[styles.BorderPosition]string {
	return map[styles.BorderPosition]string{
		styles.TopMiddleBorder: fmt.Sprintf("Command #%d", m.command.ID),
	}
}
