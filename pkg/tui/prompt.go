package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PromptMsg enables the prompt widget.
type PromptMsg struct {
	// Action to carry out when key is pressed.
	Action PromptAction
	// Key that when pressed triggers the action and closes the prompt.
	Key *key.Binding
	// Cancel is a key that when pressed skips the action and closes the prompt.
	Cancel *key.Binding
	// Prompt to display to the user.
	Prompt string
	// Set initial value for the user to edit.
	InitialValue string
	// Set placeholder text in prompt
	Placeholder string
	// CancelAnyOther, if true, checks if any key other than that specified in
	// Key is pressed. If so then the action is skipped and the prompt is
	// closed. Overrides Cancel key binding.
	CancelAnyOther bool
}

type PromptAction func(text string) tea.Cmd

type PromptStyle struct {
	ThickBorder *lipgloss.Style
	Regular     *lipgloss.Style
	PlaceHolder *lipgloss.Style
	Height      int
}

// YesNoPrompt sends a message to enable the prompt widget, specifically
// asking the user for a yes/no answer. If yes is given then the action is
// invoked.
func YesNoPrompt(prompt string, quit bool, action tea.Cmd) tea.Cmd {
	cancel := key.NewBinding(key.WithKeys("n", "N"))
	yes := key.NewBinding(key.WithKeys("y", "Y"), key.WithHelp("y", "confirm"))
	if quit {
		yes = key.NewBinding(key.WithKeys("y", "Y", "ctrl+c"), key.WithHelp("y/ctrl+c", "confirm and quit"))
	}

	return CmdHandler(PromptMsg{
		Prompt:         prompt + " (y/N): ",
		InitialValue:   "",
		Placeholder:    "",
		Cancel:         &cancel,
		CancelAnyOther: true,
		Action: func(_ string) tea.Cmd {
			return action
		},
		Key: &yes,
	})
}

func NewPrompt(msg *PromptMsg, style *PromptStyle) (*Prompt, tea.Cmd) {
	model := textinput.New()
	model.Prompt = msg.Prompt
	model.SetValue(msg.InitialValue)
	model.Placeholder = msg.Placeholder
	model.PlaceholderStyle = *style.PlaceHolder
	blink := model.Focus()

	prompt := Prompt{
		model:          model,
		action:         msg.Action,
		trigger:        msg.Key,
		cancel:         msg.Cancel,
		cancelAnyOther: msg.CancelAnyOther,
		style:          *style,
	}
	return &prompt, blink
}

// Prompt is a widget that prompts the user for input and triggers an action.
type Prompt struct {
	action         PromptAction
	style          PromptStyle
	trigger        *key.Binding
	cancel         *key.Binding
	model          textinput.Model
	cancelAnyOther bool
}

// HandleKey handles the user key press, and returns a command to be run, and
// whether the prompt should be closed.
func (p *Prompt) HandleKey(msg tea.KeyMsg) (closePrompt bool, cmd tea.Cmd) {
	switch {
	case key.Matches(msg, *p.trigger):
		cmd = p.action(p.model.Value())
		closePrompt = true
	case key.Matches(msg, *p.cancel), p.cancelAnyOther:
		cmd = ReportInfo("canceled operation")
		closePrompt = true
	default:
		p.model, cmd = p.model.Update(msg)
	}
	return
}

// HandleBlink handles the bubbletea blink message.
func (p *Prompt) HandleBlink(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Ignore key presses, they're handled by HandleKey above.
	default:
		// The blink message type is unexported so we just send unknown types to
		// the model.
		_, cmd = p.model.Update(msg)
	}
	return cmd
}

func (p *Prompt) View(width int) string {
	paddedBorder := p.style.ThickBorder
	paddedBorderWidth := paddedBorder.GetHorizontalBorderSize() + paddedBorder.GetHorizontalPadding()
	// Set available width for user entered value before it horizontally
	// scrolls.
	p.model.Width = max(0, width-lipgloss.Width(p.model.Prompt)-paddedBorderWidth)
	// Render a prompt, surrounded by a padded red border, spanning the width of the
	// terminal, accounting for width of border. Inline and MaxWidth ensures the
	// prompt remains on a single line.
	content := p.style.Regular.Inline(true).MaxWidth(width - paddedBorderWidth).Render(p.model.View())
	return paddedBorder.Width(width - paddedBorder.GetHorizontalBorderSize()).Render(content)
}

func (p *Prompt) HelpBindings() []*key.Binding {
	bindings := []*key.Binding{
		p.trigger,
	}
	if p.cancelAnyOther {
		cancel := key.NewBinding(key.WithHelp("n", "cancel"))
		bindings = append(bindings, &cancel)
	} else {
		bindings = append(bindings, p.cancel)
	}
	return bindings
}
