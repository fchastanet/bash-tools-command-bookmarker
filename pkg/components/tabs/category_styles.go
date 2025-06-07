//nolint:mnd // ignore constant for styles files
package tabs

import (
	"github.com/charmbracelet/lipgloss"
)

// CategoryTabStylesImpl implements the CategoryTabStyles interface
type CategoryTabStylesImpl struct {
	activeTabStyle       lipgloss.Style
	inactiveTabStyle     lipgloss.Style
	navigationArrowStyle lipgloss.Style
	tabCountStyle        lipgloss.Style
}

// NewCategoryTabStyles creates a new CategoryTabStylesImpl with default styles
func NewCategoryTabStyles(primaryColor lipgloss.TerminalColor) CategoryTabStyles {
	return &CategoryTabStylesImpl{
		activeTabStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("white")).
			Background(primaryColor).
			Bold(true).
			Padding(0, 2).
			Margin(0, 1),

		inactiveTabStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1).
			Margin(0, 1),

		navigationArrowStyle: lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true),

		tabCountStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("white")),
	}
}

// GetActiveTabStyle returns the style for active tabs
func (s *CategoryTabStylesImpl) GetActiveTabStyle() lipgloss.Style {
	return s.activeTabStyle
}

// GetInactiveTabStyle returns the style for inactive tabs
func (s *CategoryTabStylesImpl) GetInactiveTabStyle() lipgloss.Style {
	return s.inactiveTabStyle
}

// GetNavigationArrowStyle returns the style for navigation arrows
func (s *CategoryTabStylesImpl) GetNavigationArrowStyle() lipgloss.Style {
	return s.navigationArrowStyle
}

// GetTabCountStyle returns the style for tab counts
func (s *CategoryTabStylesImpl) GetTabCountStyle() lipgloss.Style {
	return s.tabCountStyle
}
