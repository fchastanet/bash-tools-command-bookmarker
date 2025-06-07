package top

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/internal/models"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/keys"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/structure"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/styles"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/top/footer"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/top/header"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/top/help"
	"github.com/fchastanet/shell-command-bookmarker/internal/services"
	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/internal/version"
	"github.com/fchastanet/shell-command-bookmarker/pkg/components/tabs"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

// alter how all messages are handled.
type mode int

const (
	normalMode mode = iota // default
	promptMode             // confirm prompt is visible and taking input
)

const OutputFileMode = 0o600

// QuitClearScreenMsg is a message type for quitting with screen clearing
type QuitClearScreenMsg struct{}

// QuitWithClearScreen returns a command that will quit the application with a clear screen
func QuitWithClearScreen() tea.Cmd {
	return tea.Sequence(
		tea.ExitAltScreen, // Exit alternate screen if it was used
		func() tea.Msg {
			return tea.WindowSizeMsg{Width: 0, Height: 0}
		},
		func() tea.Msg {
			return QuitClearScreenMsg{}
		},
	)
}

// Define constants for magic numbers
const (
	performanceMonitorInterval = 5 * time.Second
	bytesInMegabyte            = 1024 * 1024

	// How long messages remain displayed before auto-clearing
	messageDisplayDuration = 2 * time.Second
)

// MessageClearTickMsg represents a tick to check if messages should be cleared
type MessageClearTickMsg struct{}

type KeyMaps struct {
	filter            *tabs.FilterKeyMap
	global            *keys.GlobalKeyMap
	pane              *keys.PaneNavigationKeyMap
	tableNavigation   *table.Navigation
	tableAction       *table.Action
	tableCustomAction *keys.TableCustomActionKeyMap
	editor            *keys.EditorKeyMap
	form              *huh.KeyMap
}

type Model struct {
	// Time when the current message should be cleared
	messageClearTime time.Time

	*models.PaneManager
	appService *services.AppService
	styles     *styles.Styles
	keyMaps    *KeyMaps

	prompt *tui.YesNoPromptMsg

	spinner *spinner.Model

	selectedCommand *dbmodels.Command

	footerModel footer.Model
	headerModel header.Model
	helpModel   help.Model

	width  int
	height int
	mode   mode

	// Whether the spinner is currently active
	spinning bool

	// Performance monitoring state
	perfMonitorActive bool

	// Flag to indicate we're quitting and should clear the screen
	quitting bool
}

func NewModel(
	appService services.AppServiceInterface,
	myStyles *styles.Styles,
) Model {
	// Work-around for
	// https://github.com/charmbracelet/bubbletea/issues/1036#issuecomment-2158563056
	_ = lipgloss.HasDarkBackground()

	keyMaps := &KeyMaps{
		editor:            keys.GetDefaultEditorKeyMap(),
		filter:            keys.GetFilterKeyMap(),
		global:            keys.GetGlobalKeyMap(),
		pane:              keys.GetPaneNavigationKeyMap(),
		tableNavigation:   keys.GetTableNavigationKeyMap(),
		tableAction:       keys.GetTableActionKeyMap(),
		tableCustomAction: keys.GetTableCustomActionKeyMap(),
		form:              keys.GetFormKeyMap(),
	}

	spinnerObj := spinner.New(spinner.WithSpinner(spinner.Line))

	helpWidget := myStyles.HelpStyle.Main.Render("F1/h/alt+? help")
	versionWidget := myStyles.FooterStyle.Version.Render(version.Get())

	// Create help and footer components
	helpModel := help.New(myStyles)
	footerModel := footer.New(myStyles, helpWidget, versionWidget)

	// Create header component with application name
	headerModel := header.New(myStyles, "Shell Command Bookmarker")

	m := Model{
		PaneManager: models.NewPaneManager(
			myStyles,
			keyMaps.global,
			keyMaps.pane,
		),
		spinner:           &spinnerObj,
		appService:        appService.Self(),
		styles:            myStyles,
		keyMaps:           keyMaps,
		helpModel:         helpModel,
		footerModel:       footerModel,
		headerModel:       headerModel,
		width:             0,
		height:            0,
		mode:              normalMode,
		spinning:          false,
		prompt:            nil,
		messageClearTime:  time.Time{},
		perfMonitorActive: false,
		quitting:          false,
		selectedCommand:   nil,
	}
	makers := NewMakerFactory(m.PaneManager, appService, myStyles, &spinnerObj, keyMaps)
	m.SetMakerFactory(makers)
	return m
}

func (m *Model) Init() tea.Cmd {
	return models.SafeCmd(tea.Batch(
		m.PaneManager.Init(),
	))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Use enhanced logging for better debugging
	m.appService.LoggerService.EnhancedLogTeaMsg(msg)

	cmd := m.dispatchMessage(msg)
	return m, cmd
}

// dispatchMessage routes messages to their appropriate handlers
//
//nolint:cyclop // not really complex
func (m *Model) dispatchMessage(msg tea.Msg) tea.Cmd {
	// First handle performance monitoring messages
	if cmd := m.handlePerformanceMessages(msg); cmd != nil {
		return cmd
	}

	// Then handle other specific message types
	switch msg := msg.(type) {
	case QuitClearScreenMsg:
		m.quitting = true
		return tea.Quit
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	case tui.YesNoPromptMsg:
		return m.handleYesNoPrompt(msg)
	case tui.ErrorMsg, tui.InfoMsg, MessageClearTickMsg:
		return m.handleStatusMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case structure.FocusedPaneChangedMsg:
		return m.handleFocusPaneChangedMsg(msg)
	case cursor.BlinkMsg:
		return m.handleBlink(msg)
	case tui.MemoryStatsMsg:
		m.handleMemoryStats(msg)
		return tui.PerformanceMonitorTick(performanceMonitorInterval)
	case structure.CommandSelectedForShellMsg:
		return m.handleCommandSelectedForShellMsg(msg)
	case table.RowSelectedActionMsg[*dbmodels.Command]:
		m.selectedCommand = msg.Row
		m.updateHelpBindings()
	}

	if m.prompt != nil && m.mode == promptMode {
		return m.handlePromptMode(msg)
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKeyMsg(keyMsg)
	}
	cmd := m.PaneManager.Update(msg)
	if cmd != nil {
		return cmd
	}
	return nil
}

func (m *Model) handlePromptMode(msg tea.Msg) tea.Cmd {
	gKeys := m.keyMaps.global
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, *gKeys.Help) && gKeys.Help.Enabled() {
			m.displayHelp()
			return nil
		}
	}
	cmd := m.prompt.Update(msg)
	if m.prompt.IsCompleted() {
		// If the prompt was completed, just reset the prompt
		m.mode = normalMode
		m.prompt = nil
		m.updateHelpBindings()
		// Send out message to panes to resize themselves to remove room for the prompt
		m.PaneManager.Update(tea.WindowSizeMsg{
			Height: m.viewHeight(),
			Width:  m.viewWidth(),
		})
	}
	return cmd
}

func (m *Model) handleCommandSelectedForShellMsg(msg structure.CommandSelectedForShellMsg) tea.Cmd {
	// When a command is selected for shell, store it and quit
	if err := os.WriteFile(m.appService.Config.OutputFile, []byte(msg.Command), OutputFileMode); err != nil {
		slog.Error("Failed to write command to output file", "error", err)
		fmt.Fprintf(os.Stderr, "Error writing command to output file: %v\n", err)
	}
	m.quitting = true
	return tea.Quit
}

// handleFocusPaneChangedMsg handles the pane focus change message
func (m *Model) handleFocusPaneChangedMsg(msg tea.Msg) tea.Cmd {
	m.updateHelpBindings()
	// Send message to panes to resize themselves to make room for the prompt above it.
	slog.Debug("handleWindowSize", "viewHeight", m.viewHeight())
	_ = m.PaneManager.Update(tea.WindowSizeMsg{
		Height: m.viewHeight(),
		Width:  m.viewWidth(),
	})
	m.PaneManager.Update(msg)
	return nil
}

// handlePerformanceMessages handles performance monitoring related messages
func (m *Model) handlePerformanceMessages(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tui.PerformanceMonitorStartMsg:
		if !m.perfMonitorActive {
			m.perfMonitorActive = true
			m.footerModel.SetInfo("Performance monitoring started")
			return tui.PerformanceMonitorTick(performanceMonitorInterval)
		}
	case tui.PerformanceMonitorStopMsg:
		if m.perfMonitorActive {
			m.perfMonitorActive = false
			m.footerModel.SetInfo("Performance monitoring stopped")
			return nil
		}
	}
	return nil
}

// handleSpinnerTick processes spinner tick messages
func (m *Model) handleSpinnerTick(msg spinner.TickMsg) tea.Cmd {
	var cmd tea.Cmd
	*m.spinner, cmd = m.spinner.Update(msg)
	if m.spinning {
		// Continue spinning spinner.
		return cmd
	}
	return nil
}

// handleStatusMsg processes status messages
func (m *Model) handleStatusMsg(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tui.ErrorMsg:
		m.footerModel.SetError(error(msg))
		m.messageClearTime = time.Now().Add(messageDisplayDuration)
		cmds = append(cmds, scheduleClearMessage(messageDisplayDuration))
	case tui.InfoMsg:
		m.footerModel.SetInfo(string(msg))
		m.messageClearTime = time.Now().Add(messageDisplayDuration)
		cmds = append(cmds, scheduleClearMessage(messageDisplayDuration))
	case MessageClearTickMsg:
		// Only clear the message if the current time is after the scheduled clear time
		// This ensures a new message that resets the clear time won't be cleared prematurely
		if !m.messageClearTime.IsZero() && time.Now().After(m.messageClearTime) {
			m.footerModel.ClearMessages()
			m.messageClearTime = time.Time{}
		}
	}

	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

// scheduleClearMessage creates a tick command that triggers after the specified duration
func scheduleClearMessage(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return MessageClearTickMsg{}
	})
}

// handleWindowSize processes window size messages
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height
	m.helpModel.SetWidth(m.width)
	m.footerModel.SetWidth(m.width)
	m.headerModel.SetWidth(m.width) // Set the header width
	slog.Debug("handleWindowSize", "viewHeight", m.viewHeight())
	return m.PaneManager.Update(tea.WindowSizeMsg{
		Height: m.viewHeight(),
		Width:  m.viewWidth(),
	})
}

// handleBlink processes cursor blink messages
func (m *Model) handleBlink(msg cursor.BlinkMsg) tea.Cmd {
	return m.FocusedModel().Update(msg)
}

// handleMemoryStats processes and displays memory statistics
func (m *Model) handleMemoryStats(msg tui.MemoryStatsMsg) {
	statsInfo := fmt.Sprintf("Memory: %d MB in use | %d MB total | %d MB sys | GC runs: %d",
		msg.Alloc/bytesInMegabyte, msg.TotalAlloc/bytesInMegabyte, msg.Sys/bytesInMegabyte, msg.NumGC)

	m.footerModel.SetInfo(statsInfo)
}

// handleKeyMsg processes key messages
func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch m.mode {
	case promptMode:
		// already handled in Update method above
	case normalMode:
		cmd := m.PaneManager.Update(msg)
		if cmd != nil {
			return cmd
		}
		// If still normal mode, we let manageKey handle the key message.
		if m.mode == normalMode {
			return m.manageKey(msg)
		}
	}

	return nil
}

func (m *Model) handleYesNoPrompt(promptMsg tui.YesNoPromptMsg) tea.Cmd {
	var cmds []tea.Cmd
	m.mode = promptMode
	m.prompt = &promptMsg

	cmds = append(cmds, m.prompt.Init())
	m.updateHelpBindings()
	// Send out message to panes to resize themselves to make room for the prompt above it.
	windowSizeMsg := tea.WindowSizeMsg{
		Height: m.viewHeight(),
		Width:  m.viewWidth(),
	}
	cmds = append(
		cmds,
		m.prompt.Update(windowSizeMsg),
		m.PaneManager.Update(windowSizeMsg),
	)
	return tea.Batch(cmds...)
}

//nolint:cyclop // not really complex
func (m *Model) manageKey(msg tea.KeyMsg) tea.Cmd {
	globalKeys := m.keyMaps.global
	switch {
	case key.Matches(msg, *globalKeys.Quit) && globalKeys.Quit.Enabled():
		return m.handleQuit()
	case key.Matches(msg, *globalKeys.Help) && globalKeys.Help.Enabled():
		m.displayHelp()
	case key.Matches(msg, *globalKeys.Debug) && globalKeys.Debug.Enabled():
		// ctrl+d shows memory stats for debugging performance
		if m.perfMonitorActive {
			return tui.StopPerformanceMonitor()
		}
		return tui.StartPerformanceMonitor(performanceMonitorInterval)
	case key.Matches(msg, *globalKeys.Search) && globalKeys.Search.Enabled():
		return models.NavigateTo(structure.SearchKind, structure.WithPosition(structure.LeftPane))
	default:
	}
	return nil
}

func (m *Model) handleQuit() tea.Cmd {
	// In shell selection mode (with output-file parameter), Ctrl+C should exit immediately without confirmation
	if m.appService.IsShellSelectionMode() {
		m.quitting = true
		return QuitWithClearScreen()
	}

	// In normal mode, Ctrl+C prompts for confirmation before quitting
	return tui.YesNoPrompt(
		"Quit Shell Command Bookmarker?",
		keys.GetFormKeyMap(),
		func() tea.Cmd {
			m.quitting = true
			m.mode = normalMode
			return QuitWithClearScreen()
		},
	)
}

func (m *Model) displayHelp() {
	// '?' toggles help widget
	m.helpModel.Toggle()
	m.updateHelpBindings()
	// Help widget takes up space so update panes' dimensions
	slog.Debug("handleHelpToggle", "viewHeight", m.viewHeight())
	m.PaneManager.Update(tea.WindowSizeMsg{
		Height: m.viewHeight(),
		Width:  m.viewWidth(),
	})
}

func (m *Model) View() string {
	// When quitting, return an empty string with height 0 to clear the screen
	if m.quitting {
		return lipgloss.NewStyle().Height(0).Render("")
	}

	// Start composing vertical stack of components that fill entire terminal.
	var components []string

	// Add header with application name
	components = append(components, m.headerModel.View())

	// Add prompt if in prompt mode.
	if m.mode == promptMode {
		components = append(components, m.prompt.View())
	}
	// Add panes
	components = append(components, lipgloss.NewStyle().
		Height(m.viewHeight()).
		Width(m.viewWidth()).
		Render(m.PaneManager.View()),
	)

	// Add help if enabled
	if m.helpModel.IsVisible() {
		components = append(components, m.helpModel.View())
	}

	// Add footer
	components = append(components, m.footerModel.View())

	return strings.Join(components, "\n")
}

// updateHelpBindings updates the key bindings displayed in help based on current mode
func (m *Model) updateHelpBindings() {
	// Clear previous binding sets
	m.helpModel.ClearBindingSets()
	switch m.mode {
	case promptMode:
		// For prompt mode, just use a single set
		m.helpModel.AddBindingSet("Prompt Controls", keys.GetFormBindings())
	case normalMode:
		// For normal mode, organize bindings into logical groups
		m.helpModel.AddBindingSet("Global", keys.KeyMapToSlice(*m.keyMaps.global))
		m.helpModel.AddBindingSet("Pane Navigation", m.HelpBindings())
		if m.FocusedPosition() == structure.TopPane {
			m.helpModel.AddBindingSet("Filter Controls", keys.KeyMapToSlice(*m.keyMaps.filter))
			m.helpModel.AddBindingSet("Table Nav", keys.KeyMapToSlice(*m.keyMaps.tableNavigation))
			tableActions := m.keyMaps.tableAction
			tableCustomActions := m.keyMaps.tableCustomAction
			keys.UpdateBindings(
				tableActions,
				tableCustomActions,
				m.appService.IsShellSelectionMode(),
				m.selectedCommand,
			)
			m.helpModel.AddBindingSet("Table Actions", keys.KeyMapToSlice(*tableActions))
			m.helpModel.AddBindingSet("Command Actions", keys.KeyMapToSlice(*tableCustomActions))
		}
	}
}

// contentHeight returns the height available to the panes
func (m *Model) viewHeight() int {
	slog.Debug("viewHeight", "height", m.height)
	// Start with full height
	vh := m.height

	// Subtract header height
	vh -= m.headerModel.Height()

	// Subtract footer height
	vh -= m.footerModel.Height()

	// Subtract prompt height if in prompt mode
	if m.mode == promptMode {
		vh -= lipgloss.Height(m.prompt.View())
	}

	// Subtract help height if visible
	helpHeight := m.helpModel.Height()
	slog.Debug("viewHeight", "helpHeightIncludingBorders", helpHeight)
	vh -= helpHeight

	slog.Debug("viewHeight", "vh", vh)

	// Ensure we don't go below minimum height
	return max(m.styles.PaneStyle.MinContentHeight, vh)
}

// contentWidth returns the width available within the main view
func (m *Model) viewWidth() int {
	// Use full width but ensure we don't go below minimum width
	return max(m.styles.PaneStyle.MinContentWidth, m.width)
}
