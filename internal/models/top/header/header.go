package header

import (
	"github.com/fchastanet/shell-command-bookmarker/internal/models/styles"
)

// Model represents the header component
type Model struct {
	styles *styles.Styles
	title  string
	width  int
}

// New creates a new header component
func New(myStyles *styles.Styles, title string) Model {
	return Model{
		width:  0,
		styles: myStyles,
		title:  title,
	}
}

// Height returns the height of the header component when rendered
func (*Model) Height() int {
	return styles.HeightHeader
}

// Width returns the width of the header component
func (m *Model) Width() int {
	return m.width
}

// SetWidth updates the width of the header component
func (m *Model) SetWidth(width int) {
	m.width = width
}

// View renders the header component
func (m *Model) View() string {
	return m.styles.HeaderStyle.Title.Width(m.width).Render(m.title)
}
