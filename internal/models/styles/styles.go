package styles

import (
	"log/slog"

	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/pkg/components/tabs"
	"github.com/fchastanet/shell-command-bookmarker/pkg/sort"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/colors"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

type Styles struct {
	TableStyle     table.StyleInterface
	PaneStyle      *PaneStyle
	PlaceHolder    *lipgloss.Style
	HelpStyle      *HelpStyle
	FooterStyle    *FooterStyle
	HeaderStyle    *HeaderStyle
	WindowStyle    *WindowStyle
	EditorStyle    *EditorStyle
	ScrollbarStyle *tui.ScrollbarStyle
	// ColorTheme is the color theme used in the application.
	ColorTheme        *ColorTheme
	CategoryTabStyles tabs.CategoryTabStylesInterface
	SortStyles        sort.EditorSortStylesInterface
}

type Style struct {
	ActiveStyle   *lipgloss.Style
	InactiveStyle *lipgloss.Style
}

type FooterStyle struct {
	ErrorStyle   *lipgloss.Style
	InfoStyle    *lipgloss.Style
	DefaultStyle *lipgloss.Style
	Main         *lipgloss.Style
	Version      *lipgloss.Style
	Height       int
}

type PaneStyle struct {
	TopBorder    *lipgloss.Style
	FooterHeight int
	HeaderHeight int
	// defaultLeftPaneWidth is the default width of the left pane.
	DefaultLeftPaneWidth  int
	DefaultRightPaneWidth int
	// defaultTopPaneHeight is the default height of the top right pane.
	DefaultTopPaneHeight int
	// minimum width of each pane
	MinPaneWidth int
	// minimum height of each pane
	MinPaneHeight int
	// MinContentHeight is the minimum height of content above the footer.
	MinContentHeight int
	// MinContentWidth is the minimum width of the content.
	MinContentWidth int
	BordersWidth    int
}

type WindowStyle struct {
	BorderStyle    *lipgloss.Style
	DocStyle       *lipgloss.Style
	HighlightColor *lipgloss.AdaptiveColor
	Background     lipgloss.Color
	Foreground     lipgloss.Color
	// MinHeight is the minimum height of the TUI.
	MinHeight int
	// Height of prompt including borders
	PromptHeight int
}

type HelpStyle struct {
	Main       *lipgloss.Style
	KeyStyle   *lipgloss.Style
	DescStyle  *lipgloss.Style
	TitleStyle *lipgloss.Style // Style for binding set titles
	// Height of help widget, including borders
	Height       int
	ColumnMargin int
	BordersWidth int
}

type TableStyle struct {
	tableHeaderStyle     *lipgloss.Style
	tableHeaderCellStyle *lipgloss.Style
	// Border style for the table
	border *lipgloss.Style
	// Style for the table filters header
	filtersBlock *lipgloss.Style
	// Style for the table cell
	cell *lipgloss.Style
	// ScrollbarStyle is the style for the scrollbar.
	scrollbarStyle *tui.ScrollbarStyle

	row                   *lipgloss.Style
	currentRow            *lipgloss.Style
	selectedRow           *lipgloss.Style
	currentAndSelectedRow *lipgloss.Style
	cellEdited            *lipgloss.Style

	// Height of the table header
	headerHeight int
	// Height of filter widget
	filterHeight int
}

// EditorStyle contains styling for the command editor component
type EditorStyle struct {
	Title           *lipgloss.Style
	Label           *lipgloss.Style
	LabelFocused    *lipgloss.Style
	HelpText        *lipgloss.Style
	HelpTextFocused *lipgloss.Style
	// Added styles for readonly information
	ReadonlyLabel  *lipgloss.Style
	ReadonlyValue  *lipgloss.Style
	StatusOK       *lipgloss.Style
	StatusWarning  *lipgloss.Style
	StatusError    *lipgloss.Style
	StatusDisabled *lipgloss.Style
	ScrollbarStyle *tui.ScrollbarStyle
	ContentPadding int
}

func (*EditorStyle) GetInputWrapperWarningStyle() *lipgloss.Style {
	warningStyle := lipgloss.NewStyle().Foreground(colors.Yellow).Bold(true)
	return &warningStyle
}

func (*EditorStyle) GetTextAreaWrapperWarningStyle() *lipgloss.Style {
	warningStyle := lipgloss.NewStyle().Foreground(colors.Yellow).Bold(true)
	return &warningStyle
}

type HeaderStyle struct {
	Main   *lipgloss.Style
	Title  lipgloss.Style
	Height int
}

func NewStyles() *Styles {
	s := &Styles{
		TableStyle:        nil,
		PaneStyle:         nil,
		HelpStyle:         nil,
		FooterStyle:       nil,
		HeaderStyle:       nil,
		WindowStyle:       nil,
		EditorStyle:       nil,
		ScrollbarStyle:    nil,
		ColorTheme:        nil,
		PlaceHolder:       nil,
		CategoryTabStyles: nil,
		SortStyles:        nil,
	}

	// Initialize color theme
	colorTheme := NewDefaultColorTheme()

	// Initialize styles using the color theme
	s.ColorTheme = colorTheme

	s.CategoryTabStyles = s.getCategoryTabsStyles(colorTheme.PrimaryColor)
	s.ScrollbarStyle = tui.GetDefaultScrollbarStyle()

	s.initBaseStyles(colorTheme)
	s.initComponentStyles(colorTheme)

	return s
}

type CategoryTabStyles struct {
	activeTabStyle       *lipgloss.Style
	inactiveTabStyle     *lipgloss.Style
	navigationArrowStyle *lipgloss.Style
	tabCountStyle        *lipgloss.Style
}

func (*Styles) getCategoryTabsStyles(primaryColor lipgloss.TerminalColor) tabs.CategoryTabStylesInterface {
	activeTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white")).
		Background(primaryColor).
		Bold(true).
		Padding(0, PaddingMedium).
		Margin(0, 1)
	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Margin(0, 1)
	navigationArrowStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)
	tabCountStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white"))
	return &CategoryTabStyles{
		activeTabStyle:       &activeTabStyle,
		inactiveTabStyle:     &inactiveTabStyle,
		navigationArrowStyle: &navigationArrowStyle,
		tabCountStyle:        &tabCountStyle,
	}
}

func (s *CategoryTabStyles) GetActiveTabStyle() *lipgloss.Style {
	return s.activeTabStyle
}

func (s *CategoryTabStyles) GetInactiveTabStyle() *lipgloss.Style {
	return s.inactiveTabStyle
}

func (s *CategoryTabStyles) GetNavigationArrowStyle() *lipgloss.Style {
	return s.navigationArrowStyle
}

func (s *CategoryTabStyles) GetTabCountStyle() *lipgloss.Style {
	return s.tabCountStyle
}

func (s *Styles) initBaseStyles(colorTheme *ColorTheme) {
	// Create highlight color
	highlightColor := &lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	// Setup base styles
	regular := lipgloss.NewStyle()
	padded := regular.Padding(0, PaddingSmall)

	footerInline := regular.Inline(true)
	headerInline := regular.Inline(true)
	topBorder := regular.Border(lipgloss.NormalBorder())

	docStyle := lipgloss.NewStyle().Padding(
		PaddingSmall, PaddingMedium, PaddingSmall, PaddingMedium,
	)
	windowBorder := lipgloss.NewStyle().
		BorderForeground(highlightColor).
		Padding(0, 0).
		Align(lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		UnsetBorderTop()

	// Initialize window style
	s.WindowStyle = &WindowStyle{
		PromptHeight:   HeightPrompt,
		MinHeight:      HeightMinimum,
		BorderStyle:    &windowBorder,
		Background:     colors.EvenLighterGrey,
		Foreground:     colors.Black,
		DocStyle:       &docStyle,
		HighlightColor: highlightColor,
	}

	// Initialize pane style
	s.PaneStyle = &PaneStyle{
		DefaultLeftPaneWidth:  WidthLeftPane,
		DefaultRightPaneWidth: WidthRightPane,
		DefaultTopPaneHeight:  TopPaneHeight,
		MinPaneWidth:          WidthMinPane,
		MinPaneHeight:         HeightMinPane,
		MinContentHeight:      HeightMinimum - HeightFooter,
		MinContentWidth:       WidthMinContent,
		FooterHeight:          HeightFooter,
		HeaderHeight:          HeightHeader,
		TopBorder:             &topBorder,
		BordersWidth:          BordersWidth,
	}

	placeHolder := lipgloss.NewStyle().Faint(true)
	s.PlaceHolder = &placeHolder

	// Initialize footer style
	footerDefaultStyle := padded.Foreground(colors.Black).Background(colors.EvenLighterGrey)
	footerErrorStyle := regular.Padding(0, PaddingSmall).
		Background(colorTheme.ErrorLogLevel).
		Foreground(colors.White)
	footerInfoStyle := padded.Foreground(colors.Black).Background(colors.LightGreen)
	versionStyle := padded.Background(colors.DarkGrey).Foreground(colors.White)
	s.FooterStyle = &FooterStyle{
		Height:       HeightFooter,
		DefaultStyle: &footerDefaultStyle,
		ErrorStyle:   &footerErrorStyle,
		InfoStyle:    &footerInfoStyle,
		Main:         &footerInline,
		Version:      &versionStyle,
	}

	// Initialize header style
	headerStyle := headerInline.
		Bold(true).
		Background(colors.Blue).
		Foreground(colors.White)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center).
		Foreground(colors.White).
		Background(lipgloss.Color("#000080")) // Navy blue background

	s.HeaderStyle = &HeaderStyle{
		Height: HeightHeader,
		Main:   &headerStyle,
		Title:  titleStyle,
	}
}

func (s *Styles) initComponentStyles(colorTheme *ColorTheme) {
	// Initialize help style
	padded := lipgloss.NewStyle().Padding(0, PaddingSmall)
	bold := lipgloss.NewStyle().Bold(true)
	regular := lipgloss.NewStyle()

	// Create help style
	helpMainStyle := padded.Background(colors.Grey).Foreground(colors.White)
	helpKeyStyle := bold.Foreground(colorTheme.HelpKey).Margin(0, 1, 0, 0)
	helpDescStyle := regular.Foreground(colorTheme.HelpDesc)
	helpTitleStyle := bold.
		Foreground(colors.Blue).
		Underline(true).
		AlignHorizontal(lipgloss.Left)
	s.HelpStyle = &HelpStyle{
		Height:       HeightHelp,
		Main:         &helpMainStyle,
		KeyStyle:     &helpKeyStyle,
		DescStyle:    &helpDescStyle,
		TitleStyle:   &helpTitleStyle,
		ColumnMargin: HelpColumnMargin,
		BordersWidth: BordersWidth,
	}

	// Initialize table style
	s.TableStyle = getTableStyle(s.ScrollbarStyle)

	// Initialize editor style
	titleStyle := bold.Foreground(colors.White).Bold(true)
	labelStyle := bold.Foreground(colors.DarkGrey)
	labelStyleFocused := bold.Foreground(colors.Blue)
	helpTextStyle := regular.Foreground(colors.Grey)
	helpTextStyleFocused := regular.Bold(true)
	readonlyLabelStyle := bold.Foreground(colors.LightGrey)
	readonlyValueStyle := regular.Foreground(colors.LightGrey)
	statusOKStyle := regular.Foreground(colors.Green)
	statusWarningStyle := regular.Foreground(colors.Yellow)
	statusErrorStyle := regular.Foreground(colors.Red)
	statusDisabledStyle := regular.Foreground(colors.DarkGrey)

	s.EditorStyle = &EditorStyle{
		Title:           &titleStyle,
		Label:           &labelStyle,
		LabelFocused:    &labelStyleFocused,
		HelpText:        &helpTextStyle,
		HelpTextFocused: &helpTextStyleFocused,
		ContentPadding:  PaddingSmall,
		ReadonlyLabel:   &readonlyLabelStyle,
		ReadonlyValue:   &readonlyValueStyle,
		StatusOK:        &statusOKStyle,
		StatusWarning:   &statusWarningStyle,
		StatusError:     &statusErrorStyle,
		StatusDisabled:  &statusDisabledStyle,
		ScrollbarStyle:  s.ScrollbarStyle,
	}

	s.SortStyles = getEditorSortStyles()
}

func getTableStyle(scrollbarStyle *tui.ScrollbarStyle) table.StyleInterface {
	regular := lipgloss.NewStyle()

	CurrentBackground := colors.Grey
	CurrentForeground := colors.White
	SelectedBackground := lipgloss.Color("110")
	SelectedForeground := colors.Black
	CurrentAndSelectedBackground := lipgloss.Color("117")
	CurrentAndSelectedForeground := colors.Black

	tableHeaderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true)

	tableHeaderCellStyle := lipgloss.NewStyle().
		Inline(true).
		Bold(true).
		Italic(true)

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

	return &TableStyle{
		tableHeaderStyle:      &tableHeaderStyle,
		tableHeaderCellStyle:  &tableHeaderCellStyle,
		border:                &tableBorderStyle,
		filtersBlock:          &tableFilterBlock,
		cell:                  &tableCell,
		cellEdited:            &cellEdited,
		row:                   &row,
		currentRow:            &currentRow,
		selectedRow:           &selectedRow,
		currentAndSelectedRow: &currentAndSelectedRow,
		headerHeight:          TableHeaderHeight,
		filterHeight:          TableFilterHeight,
		scrollbarStyle:        scrollbarStyle,
	}
}

func (t *TableStyle) GetTableBorderStyle() *lipgloss.Style {
	return t.border
}

func (t *TableStyle) GetTableFiltersBlockStyle() *lipgloss.Style {
	return t.filtersBlock
}

func (t *TableStyle) GetTableCellStyle() *lipgloss.Style {
	return t.cell
}

func (t *TableStyle) GetTableRowStyle() *lipgloss.Style {
	return t.row
}

func (t *TableStyle) GetTableCurrentRowStyle() *lipgloss.Style {
	return t.currentRow
}

func (t *TableStyle) GetTableSelectedRowStyle() *lipgloss.Style {
	return t.selectedRow
}

func (t *TableStyle) GetTableCurrentAndSelectedRowStyle() *lipgloss.Style {
	return t.currentAndSelectedRow
}

func (t *TableStyle) GetTableCellEditedStyle() *lipgloss.Style {
	return t.cellEdited
}

func (t *TableStyle) GetTableScrollbarStyle() *tui.ScrollbarStyle {
	return t.scrollbarStyle
}

func (t *TableStyle) GetTableHeaderHeight() int {
	return t.headerHeight
}

func (t *TableStyle) GetTableFilterHeight() int {
	return t.filterHeight
}

func (t *TableStyle) GetTableHeaderStyle() *lipgloss.Style {
	return t.tableHeaderStyle
}

func (t *TableStyle) GetTableHeaderCellStyle() *lipgloss.Style {
	return t.tableHeaderCellStyle
}

func (s *Styles) Init() {
	s.checkDimension()
}

func (s *Styles) checkDimension() {
	slog.Debug("checking dimensions of styles",
		"minContentHeight", s.PaneStyle.MinContentHeight,
		"minPaneHeight", s.PaneStyle.MinPaneHeight,
		"minHelpHeight", MinHelpHeight,
	)
	if s.PaneStyle.MinPaneHeight+MinHelpHeight > s.PaneStyle.MinContentHeight {
		panic("minimum heights of panes, prompt, footer, and help cannot exceed overall minimum height")
	}
	slog.Debug("checking dimensions of styles",
		"minPaneWidth", s.PaneStyle.MinPaneWidth,
		"minContentWidth", s.PaneStyle.MinContentWidth,
	)
	if s.PaneStyle.MinPaneWidth*2 > s.PaneStyle.MinContentWidth {
		panic("minimum width of panes must be no more than half of the minimum content width")
	}
	slog.Debug("checking dimensions of styles",
		"minPaneHeight", s.PaneStyle.MinPaneHeight,
		"minTopRightPaneHeight", s.PaneStyle.DefaultTopPaneHeight,
	)
	if s.PaneStyle.MinPaneHeight > s.PaneStyle.DefaultTopPaneHeight {
		panic("default top right pane height must not be lower than the overall minimum height")
	}
	slog.Debug("checking dimensions of styles",
		"minPaneWidth", s.PaneStyle.MinPaneWidth,
		"minRightPaneWidth", s.PaneStyle.DefaultLeftPaneWidth,
	)
	if s.PaneStyle.MinPaneWidth > s.PaneStyle.DefaultLeftPaneWidth {
		panic("default left pane width must not be lower than the overall minimum width")
	}
}

func (s *EditorStyle) GetScrollbarThumb() string {
	return s.ScrollbarStyle.Thumb
}

func (s *EditorStyle) GetScrollbarTrack() string {
	return s.ScrollbarStyle.Track
}

func (s *EditorStyle) GetScrollbarWidth() int {
	return s.ScrollbarStyle.Width
}

func getEditorSortStyles() sort.EditorSortStylesInterface {
	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	return Style{
		ActiveStyle:   &activeStyle,
		InactiveStyle: &inactiveStyle,
	}
}

func (s Style) GetActiveStyle() *lipgloss.Style {
	return s.ActiveStyle
}

func (s Style) GetInactiveStyle() *lipgloss.Style {
	return s.InactiveStyle
}
