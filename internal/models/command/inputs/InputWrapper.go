package inputs

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

// InputWrapper wraps textinput.Model to implement the Input interface
type InputWrapper struct {
	Model    *textinput.Model
	style    InputWrapperStyle
	readOnly bool
}

type InputWrapperStyle interface {
	GetInputWrapperWarningStyle() *lipgloss.Style
}

func NewInputWrapper(
	placeHolder string,
	style InputWrapperStyle,
) *InputWrapper {
	textInput := textinput.New()
	textInput.Placeholder = placeHolder
	return &InputWrapper{
		Model:    &textInput,
		readOnly: false,
		style:    style,
	}
}

func (w *InputWrapper) SetCharLimit(charLimit int) {
	w.Model.CharLimit = charLimit
}

// Update implements the Input interface
func (w *InputWrapper) Update(msg tea.Msg) (Input, tea.Cmd) {
	if w.readOnly {
		return w, nil
	}
	newModel, cmd := w.Model.Update(msg)
	w.Model = &newModel
	return w, cmd
}

func (w *InputWrapper) SetReadOnly(readOnly bool) {
	w.readOnly = readOnly
}

func (w *InputWrapper) SetWidth(width int) {
	w.Model.Width = width
}

func (*InputWrapper) SetHeight(_ int) {
	// do nothing
}

func (w *InputWrapper) Blur() {
	w.Model.Blur()
}

func (w *InputWrapper) Focus() tea.Cmd {
	return w.Model.Focus()
}

func (w *InputWrapper) Value() string {
	return w.Model.Value()
}

func (w *InputWrapper) SetValue(value string) {
	w.Model.SetValue(value)
}

func (w *InputWrapper) View() string {
	if w.readOnly {
		return w.Model.Value()
	}
	txt := w.Model.View()
	if !w.readOnly && w.Model.CharLimit > 0 {
		length := len(w.Model.Value())
		availSpace := w.Model.CharLimit - length
		if availSpace <= 0 {
			warningMsg := w.style.GetInputWrapperWarningStyle().Render("No more characters can be added, limit reached.")
			txt += "\n" + warningMsg
		} else {
			txt += fmt.Sprintf("\nLength: %d/%d", length, w.Model.CharLimit)
		}
	}
	return txt
}
