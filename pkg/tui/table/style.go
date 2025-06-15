package table

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
)

const (
	PaddingSmall = 1
)

type StyleInterface interface {
	GetTableHeaderStyle() *lipgloss.Style
	GetTableHeaderCellStyle() *lipgloss.Style
	GetTableBorderStyle() *lipgloss.Style
	GetTableFiltersBlockStyle() *lipgloss.Style
	GetTableCellStyle() *lipgloss.Style
	GetTableRowStyle() *lipgloss.Style
	GetTableCurrentRowStyle() *lipgloss.Style
	GetTableSelectedRowStyle() *lipgloss.Style
	GetTableCurrentAndSelectedRowStyle() *lipgloss.Style
	GetTableCellEditedStyle() *lipgloss.Style
	GetTableScrollbarStyle() *tui.ScrollbarStyle
	GetTableHeaderHeight() int
	GetTableFilterHeight() int
}
