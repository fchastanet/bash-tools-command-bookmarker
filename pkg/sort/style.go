package sort

import "github.com/charmbracelet/lipgloss"

type EditorSortStylesInterface interface {
	GetActiveStyle() *lipgloss.Style
	GetInactiveStyle() *lipgloss.Style
}
