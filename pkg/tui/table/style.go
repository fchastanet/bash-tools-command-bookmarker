package table

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/colors"
)

const (
	PaddingSmall = 1

	// Height constants
	FilterHeight = 2
	HeaderHeight = 2
)

type Style struct {
	// Border style for the table
	Border *lipgloss.Style
	// Style for the table filters header
	FiltersBlock *lipgloss.Style
	// Style for the table cell
	Cell *lipgloss.Style
	// ScrollbarStyle is the style for the scrollbar.
	ScrollbarStyle *tui.ScrollbarStyle

	Row                   *lipgloss.Style
	CurrentRow            *lipgloss.Style
	SelectedRow           *lipgloss.Style
	CurrentAndSelectedRow *lipgloss.Style
	CellEdited            *lipgloss.Style

	// Height of the table header
	HeaderHeight int
	// Height of filter widget
	FilterHeight int
}

func GetDefaultStyle(scrollbarStyle *tui.ScrollbarStyle) *Style {
	regular := lipgloss.NewStyle()

	CurrentBackground := colors.Grey
	CurrentForeground := colors.White
	SelectedBackground := lipgloss.Color("110")
	SelectedForeground := colors.Black
	CurrentAndSelectedBackground := lipgloss.Color("117")
	CurrentAndSelectedForeground := colors.Black

	tableFilterBlock := regular.Margin(0, 1)
	tableCell := regular.Padding(0, PaddingSmall)
	tableBorderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	cellEdited := regular.Italic(true).Foreground(colors.Yellow)

	// Row styles using the color theme
	row := lipgloss.NewStyle()
	currentRow := tableCell.
		Background(CurrentBackground).
		Foreground(CurrentForeground)
	selectedRow := tableCell.
		Background(SelectedBackground).
		Foreground(SelectedForeground)
	currentAndSelectedRow := tableCell.
		Background(CurrentAndSelectedBackground).
		Foreground(CurrentAndSelectedForeground)

	return &Style{
		Border:                &tableBorderStyle,
		FiltersBlock:          &tableFilterBlock,
		Cell:                  &tableCell,
		CellEdited:            &cellEdited,
		Row:                   &row,
		CurrentRow:            &currentRow,
		SelectedRow:           &selectedRow,
		CurrentAndSelectedRow: &currentAndSelectedRow,
		HeaderHeight:          HeaderHeight,
		FilterHeight:          FilterHeight,
		ScrollbarStyle:        scrollbarStyle,
	}
}

func (s *Style) GetTableBorderStyle() *lipgloss.Style {
	return s.Border
}
