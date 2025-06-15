package tabs

import (
	"github.com/charmbracelet/lipgloss"
)

// CategoryTabStylesImpl implements the CategoryTabStyles interface
type CategoryTabStylesInterface interface {
	GetActiveTabStyle() *lipgloss.Style
	GetInactiveTabStyle() *lipgloss.Style
	GetNavigationArrowStyle() *lipgloss.Style
	GetTabCountStyle() *lipgloss.Style
}
