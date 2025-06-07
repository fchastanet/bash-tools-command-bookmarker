package filters

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel defines the interface for a filter component
type InputModel interface {
	GetFilterValue() string
	SetFilterValue(string)
	Focus() tea.Cmd
	Blur()
	SetWidth(int)
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
}

// Input is a wrapper around textinput.Model to implement the InputModel interface
type Input struct {
	textinput textinput.Model
	active    bool
}

// NewInput creates a new Input instance
func NewInput() *Input {
	ti := textinput.New()
	ti.Prompt = "Filter: "
	return &Input{
		textinput: ti,
		active:    false,
	}
}

// Init implements the tea.Model interface
func (f *Input) Init() tea.Cmd {
	return nil
}

// Update implements the tea.Model interface
func (f *Input) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.textinput, cmd = f.textinput.Update(msg)
	return cmd
}

// View implements the tea.Model interface
func (f *Input) View() string {
	return f.textinput.View()
}

// GetFilterValue returns the current filter value
func (f *Input) GetFilterValue() string {
	return f.textinput.Value()
}

// SetFilterValue sets the filter value
func (f *Input) SetFilterValue(value string) {
	f.textinput.SetValue(value)
	f.active = value != ""
}

// Focus implements the InputModel interface
func (f *Input) Focus() tea.Cmd {
	return f.textinput.Focus()
}

// Blur implements the InputModel interface
func (f *Input) Blur() {
	f.textinput.Blur()
}

// SetWidth sets the width of the filter component
func (f *Input) SetWidth(width int) {
	f.textinput.Width = width
}

// Focused returns whether the filter is focused
func (f *Input) Focused() bool {
	return f.textinput.Focused()
}

// Value returns the current value of the filter
func (f *Input) Value() string {
	return f.textinput.Value()
}
