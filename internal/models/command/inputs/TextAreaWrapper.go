package inputs

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// TextAreaWrapper wraps textarea.Model to implement the Input interface
type TextAreaWrapper struct {
	*textarea.Model
	readOnly         bool
	markdownRenderer *glamour.TermRenderer
}

type TextAreaWrapperOption func(*TextAreaWrapper) error

func NewTextAreaWrapper(
	height int, placeHolder string,
	options ...TextAreaWrapperOption,
) *TextAreaWrapper {
	textArea := textarea.New()
	textArea.Placeholder = placeHolder
	textArea.SetHeight(height)

	wrapper := &TextAreaWrapper{
		Model:            &textArea,
		readOnly:         false,
		markdownRenderer: nil,
	}

	for _, opt := range options {
		if err := opt(wrapper); err != nil {
			slog.Error("failed to apply TextAreaWrapper option", "error", err)
		}
	}

	return wrapper
}

func WithMarkdown(markdownWordWrapWidth int) TextAreaWrapperOption {
	return func(tr *TextAreaWrapper) error {
		slog.Info("Markdown rendering enabled for TextAreaWrapper")
		r, _ := glamour.NewTermRenderer(
			// detect background color and pick either the default dark or light theme
			glamour.WithAutoStyle(),
			// wrap output at specific width (default is 80)
			glamour.WithWordWrap(markdownWordWrapWidth),
		)
		tr.markdownRenderer = r
		return nil
	}
}

func (w *TextAreaWrapper) SetCharLimit(charLimit int) {
	w.CharLimit = charLimit
	w.CharLimit = charLimit
}

// Update implements the Input interface
func (w *TextAreaWrapper) Update(msg tea.Msg) (Input, tea.Cmd) {
	newModel, cmd := w.Model.Update(msg)
	w.Model = &newModel
	return w, cmd
}

// Blur implements the Input interface
func (w *TextAreaWrapper) Blur() {
	w.Model.Blur()
}

// Focus implements the Input interface
func (w *TextAreaWrapper) Focus() tea.Cmd {
	return w.Model.Focus()
}

// SetValue implements the Input interface
func (w *TextAreaWrapper) SetValue(value string) {
	w.Model.SetValue(value)
}

// Value implements the Input interface
func (w *TextAreaWrapper) Value() string {
	return w.Model.Value()
}

// View implements the Input interface
func (w *TextAreaWrapper) View() string {
	value := w.Model.Value()
	if w.readOnly && w.markdownRenderer != nil && value != "" {
		text, err := w.markdownRenderer.Render(value)
		if err != nil {
			slog.Error("failed to render markdown", "error", err)
			return value // Fallback to plain text if rendering fails
		}
		return text
	}
	txt := w.Model.View()
	if !w.readOnly && w.CharLimit > 0 {
		txt += fmt.Sprintf("\nLength: %d/%d\n", w.Length(), w.CharLimit)
	}
	return txt
}

// SetWidth implements the Input interface
func (w *TextAreaWrapper) SetWidth(width int) {
	w.Model.SetWidth(width)
}

// SetHeight implements the Input interface
func (w *TextAreaWrapper) SetHeight(height int) {
	w.Model.SetHeight(height)
}

// SetReadOnly implements the Input interface
func (w *TextAreaWrapper) SetReadOnly(readOnly bool) {
	w.readOnly = readOnly
}
