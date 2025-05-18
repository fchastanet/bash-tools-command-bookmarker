package top

import (
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
)

func (m *Model) handlePerformanceStart() (tea.Cmd, tui.PropagationMsgInterface) {
	if !m.perfMonitorActive {
		m.perfMonitorActive = true
		m.footerModel.SetInfo("Performance monitoring started")
		return tui.PerformanceMonitorTick(performanceMonitorInterval), nil
	}
	return nil, nil
}

func (m *Model) handlePerformanceStop() (tea.Cmd, tui.PropagationMsgInterface) {
	if m.perfMonitorActive {
		m.perfMonitorActive = false
		m.footerModel.SetInfo("Performance monitoring stopped")
		return nil, nil
	}
	return nil, nil
}

// handleSpinnerTick processes spinner tick messages
func (m *Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Cmd, tui.PropagationMsgInterface) {
	// If the spinner is not active, return immediately
	var cmd tea.Cmd
	*m.spinner, cmd = m.spinner.Update(msg)
	if m.spinning {
		// Continue spinning spinner.
		return cmd, nil
	}
	return nil, nil
}

// handlePrompt processes prompt messages
func (m *Model) handlePrompt(msg *tui.PromptMsg) (tea.Cmd, tui.PropagationMsgInterface) {
	// Enable prompt widget
	m.mode = promptMode
	var blink tea.Cmd
	m.prompt, blink = tui.NewPrompt(msg, m.styles.PromptStyle)
	// Send out message to panes to resize themselves to make room for the prompt above it.
	slog.Debug("handlePrompt", "viewHeight", m.viewHeight())
	return blink, &tui.PropagationMsg{
		PropagationFilter: tui.PropagationToAllChildren,
		Msg: tea.WindowSizeMsg{
			Height: m.viewHeight(),
			Width:  m.viewWidth(),
		},
	}
}

func (m *Model) handleError(msg tui.ErrorMsg) (tea.Cmd, tui.PropagationMsgInterface) {
	m.footerModel.SetError(error(msg))
	m.messageClearTime = m.messageClearTime.Add(messageDisplayDuration)
	return scheduleClearMessage(messageDisplayDuration), nil
}

func (m *Model) handleInfo(msg tui.InfoMsg) (tea.Cmd, tui.PropagationMsgInterface) {
	m.footerModel.SetInfo(string(msg))
	m.messageClearTime = time.Now().Add(messageDisplayDuration)
	return scheduleClearMessage(messageDisplayDuration), nil
}

func (m *Model) handleMessageClearTick() (tea.Cmd, tui.PropagationMsgInterface) {
	// Only clear the message if the current time is after the scheduled clear time
	// This ensures a new message that resets the clear time won't be cleared prematurely
	if !m.messageClearTime.IsZero() && time.Now().After(m.messageClearTime) {
		m.footerModel.ClearMessages()
		m.messageClearTime = time.Time{}
	}
	return nil, nil
}

// handleWindowSize processes window size messages
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Cmd, tui.PropagationMsgInterface) {
	m.width = msg.Width
	m.height = msg.Height
	m.helpModel.SetWidth(m.width)
	m.footerModel.SetWidth(m.width)
	m.headerModel.SetWidth(m.width) // Set the header width
	slog.Debug("handleWindowSize", "viewHeight", m.viewHeight())
	return nil, &tui.PropagationMsg{
		PropagationFilter: tui.PropagationToAllChildren,
		Msg: tea.WindowSizeMsg{
			Height: m.viewHeight(),
			Width:  m.viewWidth(),
		},
	}
}

// handleFocusPaneChangedMsg handles the pane focus change message
func (m *Model) handleFocusPaneChangedMsg(_ tea.Msg) (tea.Cmd, tui.PropagationMsgInterface) {
	// Update the help bindings when the focused pane changes
	m.updateHelpBindings()
	// Send message to panes to resize themselves to make room for the prompt above it.
	slog.Debug("handleWindowSize", "viewHeight", m.viewHeight())
	return nil, &tui.PropagationMsg{
		PropagationFilter: tui.PropagationToFocusedChildren,
		Msg: tea.WindowSizeMsg{
			Height: m.viewHeight(),
			Width:  m.viewWidth(),
		},
	}
}

// handleBlink processes cursor blink messages
func (m *Model) handleBlink(msg cursor.BlinkMsg) (tea.Cmd, tui.PropagationMsgInterface) {
	// Send blink message to prompt if in prompt mode
	if m.mode == promptMode {
		return m.prompt.HandleBlink(msg), nil
	}
	// otherwise forward it to the active pane to handle.
	return nil, &tui.PropagationMsg{
		PropagationFilter: tui.PropagationToFocusedChildren,
		Msg:               msg,
	}
}
