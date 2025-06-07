package table

import (
	"log/slog"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/utils"
	"golang.org/x/exp/maps"
)

const HalfPageMultiplier = 2

type EditorInterface interface {
	EditionInProgress() bool
}

type EditorsCacheInterface interface {
	// Get retrieves a model from the cache.
	Get(resource.ID) EditorInterface
}

// Model defines a state for the table widget.
type Model[V resource.Identifiable] struct {
	borderColor lipgloss.TerminalColor

	previewKind      resource.Kind
	styles           *Style
	navigationKeyMap *Navigation
	actionKeyMap     *Action
	rowRenderer      RowRenderer[V]
	cellRenderer     DynamicCellRenderer[V]
	rendered         map[resource.ID]RenderedRow
	editorsCache     EditorsCacheInterface

	// items are the unfiltered set of items available to the table.
	items    map[resource.ID]V
	sortFunc SortFunc[V]

	selected map[resource.ID]V

	border lipgloss.Border
	cols   []Column
	rows   []V

	currentRowIndex int
	currentRowID    resource.ID

	// index of first visible row
	start int

	// width of table without borders
	width int
	// height of table without borders
	height int

	focused    bool
	selectable bool
}

// Column defines the table structure.
type Column struct {
	TruncationFunc func(s string, w int, tail string) string
	Key            ColumnKey
	// TODO: Default to upper case of key
	Title      string
	Width      int
	FlexFactor int
	// RightAlign aligns content to the right. If false, content is aligned to
	// the left.
	RightAlign bool
}

type ColumnKey string

type RowRenderer[V any] func(V) RenderedRow

type DynamicCellRenderer[V any] func(row V, cellContent string, colIndex int, rowEdited bool) string

// RenderedRow provides the rendered string for each column in a row.
type RenderedRow map[ColumnKey]string

type SortFunc[V any] func(V, V) int

// BulkInsertMsg performs a bulk insertion of entities into a table
type BulkInsertMsg[T any] struct {
	InfoMsg     string
	Items       []T
	SelectRowID resource.ID
}

// New creates a new model for the table widget.
func New[V resource.Identifiable](
	editorsCache EditorsCacheInterface,
	tableStyles *Style,
	cols []Column,
	rowRenderer RowRenderer[V],
	cellRenderer DynamicCellRenderer[V],
	width, height int, opts ...Option[V],
) Model[V] {
	m := Model[V]{
		focused:          false,
		styles:           tableStyles,
		editorsCache:     editorsCache,
		navigationKeyMap: nil,
		actionKeyMap:     nil,
		cols:             make([]Column, len(cols)),
		rows:             []V{},
		rowRenderer:      rowRenderer,
		cellRenderer:     cellRenderer,
		items:            make(map[resource.ID]V),
		rendered:         make(map[resource.ID]RenderedRow),
		selected:         make(map[resource.ID]V),
		selectable:       true,
		border:           lipgloss.NormalBorder(),
		borderColor:      lipgloss.NoColor{},
		currentRowIndex:  -1,
		currentRowID:     resource.ID(0),
		sortFunc:         nil,
		start:            0,
		width:            width,
		height:           height,
		previewKind:      resource.DefaultKind{},
	}
	for _, fn := range opts {
		fn(&m)
	}
	if m.navigationKeyMap == nil {
		m.navigationKeyMap = GetDefaultNavigation()
	}
	if m.actionKeyMap == nil {
		m.actionKeyMap = GetDefaultAction()
	}

	// Copy column structs onto receiver, because the caller may modify columns.
	copy(m.cols, cols)
	// For each column, set default truncation function if unset.
	for i, col := range m.cols {
		if col.TruncationFunc == nil {
			m.cols[i].TruncationFunc = GetDefaultTruncationFunc()
		}
	}

	m.setDimensions(width, height)

	return m
}

type Option[V resource.Identifiable] func(m *Model[V])

// WithSortFunc configures the table to sort rows using the given func.
func WithSortFunc[V resource.Identifiable](sortFunc func(V, V) int) Option[V] {
	return func(m *Model[V]) {
		m.sortFunc = sortFunc
	}
}

// WithNavigation configures the table to use the given navigation keys.
func WithNavigation[V resource.Identifiable](nav *Navigation) Option[V] {
	return func(m *Model[V]) {
		m.navigationKeyMap = nav
	}
}

func WithAction[V resource.Identifiable](action *Action) Option[V] {
	return func(m *Model[V]) {
		m.actionKeyMap = action
	}
}

// WithSelectable sets whether rows are selectable.
func WithSelectable[V resource.Identifiable](s bool) Option[V] {
	return func(m *Model[V]) {
		m.selectable = s
	}
}

// WithPreview configures the table to automatically populate the bottom right
// pane with a model corresponding to the current row.
func WithPreview[V resource.Identifiable](kind resource.Kind) Option[V] {
	return func(m *Model[V]) {
		m.previewKind = kind
	}
}

func (*Model[V]) BeforeSwitchPane() tea.Cmd {
	return nil
}

func (m *Model[V]) IsFocused() bool {
	return m.focused
}

func (m *Model[V]) Focus() {
	m.focused = true
}

func (m *Model[V]) Blur() {
	m.focused = false
}

func (m *Model[V]) SetRows(rows []V) {
	m.rows = rows
}

func (m *Model[V]) SetColumns(columns []Column) {
	m.cols = columns
}

func (m *Model[V]) SetWidth(width int) {
	m.width = width
}

func (m *Model[V]) SetHeight(height int) {
	m.height = height
}

// setDimensions sets the dimensions of the table.
func (m *Model[V]) setDimensions(width, height int) {
	m.height = height
	m.width = width
	m.setColumnWidths()

	m.setStart()
}

// rowAreaHeight returns the height of the terminal allocated to rows.
func (m *Model[V]) rowAreaHeight() int {
	height := max(0, m.height-m.styles.HeaderHeight)

	slog.Debug("table rowAreaHeight", "height", height)
	return height
}

// visibleRows returns the number of visible rows that can be
// rendered in the available space.
func (m *Model[V]) visibleRows() int {
	// The number of visible rows cannot exceed the row area height.
	return min(m.rowAreaHeight(), len(m.rows)-m.start)
}

// Update is the Bubble Tea update loop.
func (m *Model[V]) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		m.setDimensions(msg.Width, msg.Height)
		return nil
	case resource.Event[V]:
		return m.handleResourceEvent(msg)
	case BulkInsertMsg[V]:
		return m.handleBulkInsert(msg)
	}
	return nil
}

func (m *Model[V]) handleBulkInsert(msg BulkInsertMsg[V]) tea.Cmd {
	m.SetItems(msg.Items...)
	if msg.SelectRowID != resource.ID(0) {
		// If a specific row ID is provided, select that row.
		cmd := m.GotoID(msg.SelectRowID)
		if cmd != nil {
			return cmd
		}
		// row ID is not found, keep previous selection if possible
	}
	var rowCmd tea.Cmd
	if len(msg.Items) == 0 {
		m.currentRowIndex = -1
		rowCmd = tui.CmdHandler(RowSelectedActionMsg[V]{
			Row:   *new(V),
			RowID: resource.ID(0),
		})
	} else {
		// should we keep the current row index?
		if m.currentRowIndex >= len(m.rows) {
			m.currentRowIndex = 0
		}
		rowCmd = tui.CmdHandler(RowSelectedActionMsg[V]{
			Row:   m.rows[m.currentRowIndex],
			RowID: m.currentRowID,
		})
	}
	return tea.Batch(
		rowCmd,
		tui.ReportInfo(msg.InfoMsg),
	)
}

func (m *Model[V]) handleResourceEvent(msg resource.Event[V]) tea.Cmd {
	switch msg.Type {
	case resource.CreatedEvent, resource.UpdatedEvent:
		m.AddItems(msg.Payload)
	case resource.DeletedEvent:
		m.removeItem(msg.Payload)
	}
	return nil
}

// handleKeyMsg processes key press messages
func (m *Model[V]) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Group navigation keys
	if cmd := m.handleNavigationKey(msg); cmd != nil {
		return cmd
	}

	// Group selection keys
	if m.handleSelectionKey(msg) {
		return nil
	}

	// Handle action keys
	if cmd := m.handleActionKey(msg); cmd != nil {
		return cmd
	}

	return nil
}

func (m *Model[V]) handleActionKey(msg tea.KeyMsg) tea.Cmd {
	actions := m.actionKeyMap
	switch {
	case key.Matches(msg, *actions.Enter) && actions.Enter.Enabled():
		row, ok := m.CurrentRow()
		if !ok {
			return nil
		}
		return tui.CmdHandler(RowDefaultActionMsg[V]{
			Row:   row,
			RowID: row.GetID(),
			Kind:  m.previewKind,
		})
	case key.Matches(msg, *actions.Reload) && actions.Reload.Enabled():
		return tui.CmdHandler(ReloadMsg[V]{
			RowID: -1, InfoMsg: nil,
		})
	case key.Matches(msg, *actions.Delete) && actions.Delete.Enabled():
		row, ok := m.CurrentRow()
		if !ok {
			return nil
		}
		return tui.CmdHandler(RowDeleteActionMsg[V]{
			Row:   row,
			RowID: row.GetID(),
		})
	}
	return nil
}

// handleNavigationKey handles all navigation key presses
//
//nolint:cyclop // not really complex
func (m *Model[V]) handleNavigationKey(msg tea.KeyMsg) tea.Cmd {
	nav := m.navigationKeyMap
	switch {
	case key.Matches(msg, *nav.LineUp) && nav.LineUp.Enabled():
		m.MoveUp(1)
	case key.Matches(msg, *nav.LineDown) && nav.LineDown.Enabled():
		m.MoveDown(1)
	case key.Matches(msg, *nav.PageUp) && nav.PageUp.Enabled():
		m.MoveUp(m.rowAreaHeight())
	case key.Matches(msg, *nav.PageDown) && nav.PageDown.Enabled():
		m.MoveDown(m.rowAreaHeight())
	case key.Matches(msg, *nav.HalfPageUp) && nav.HalfPageUp.Enabled():
		m.MoveUp(m.rowAreaHeight() / HalfPageMultiplier)
	case key.Matches(msg, *nav.HalfPageDown) && nav.HalfPageDown.Enabled():
		m.MoveDown(m.rowAreaHeight() / HalfPageMultiplier)
	case key.Matches(msg, *nav.GotoTop) && nav.GotoTop.Enabled():
		m.GotoTop()
	case key.Matches(msg, *nav.GotoBottom) && nav.GotoBottom.Enabled():
		m.GotoBottom()
	default:
		return nil
	}

	return tui.CmdHandler(RowSelectedActionMsg[V]{
		Row:   m.rows[m.currentRowIndex],
		RowID: m.currentRowID,
	})
}

// handleSelectionKey handles all selection key presses
func (m *Model[V]) handleSelectionKey(msg tea.KeyMsg) bool {
	nav := m.actionKeyMap
	switch {
	case key.Matches(msg, *nav.Select) && nav.Select.Enabled():
		m.ToggleSelection()
	case key.Matches(msg, *nav.SelectAll) && nav.SelectAll.Enabled():
		m.SelectAll()
	case key.Matches(msg, *nav.SelectClear) && nav.SelectClear.Enabled():
		m.DeselectAll()
	case key.Matches(msg, *nav.SelectRange) && nav.SelectRange.Enabled():
		m.SelectRange()
	default:
		return false
	}
	return true
}

// PreviewCurrentRow returns information for previewing the current row
func (m *Model[V]) PreviewCurrentRow() (
	resourceKind resource.Kind,
	resourceID resource.ID,
	previewAvailable bool,
) {
	resourceKind = m.previewKind
	previewAvailable = false
	if _, ok := m.CurrentRow(); ok {
		resourceID = m.currentRowID
		previewAvailable = true
	}
	return resourceKind, resourceID, previewAvailable
}

// View renders the table.
func (m *Model[V]) View() string {
	// Table is composed of a vertical stack of components:
	// (a) optional filter widget
	// (b) header
	// (c) rows + scrollbar
	//
	// TODO: this allocation logic is wrong
	components := make([]string, 0, 1+1+m.visibleRows())
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true)
	components = append(components, style.Render(m.headersView()))
	// Generate scrollbar
	scrollbar := tui.Scrollbar(
		m.styles.ScrollbarStyle,
		m.rowAreaHeight(), len(m.rows), m.visibleRows(), m.start)
	// Get all the visible rows
	rows := make([]string, 0, m.visibleRows())
	for i := range m.visibleRows() {
		rows = append(rows, m.renderRow(m.start+i))
	}
	rowArea := lipgloss.NewStyle().
		Width(m.width - m.styles.ScrollbarStyle.Width).
		Render(strings.Join(rows, "\n"))
	// Put rows alongside the scrollbar to the right.
	components = append(components, lipgloss.JoinHorizontal(lipgloss.Top, rowArea, scrollbar))
	// Render table components, ensuring it is at least a min height
	content := lipgloss.NewStyle().
		Height(m.height).
		MaxHeight(m.height).
		Render(lipgloss.JoinVertical(lipgloss.Top, components...))
	return content
}

// CurrentRow returns the current row the user has highlighted.  If the table is
// empty then false is returned.
func (m *Model[V]) CurrentRow() (V, bool) {
	if m.currentRowIndex < 0 || m.currentRowIndex >= len(m.rows) {
		return *new(V), false
	}
	return m.rows[m.currentRowIndex], true
}

// SelectedOrCurrent returns either the selected rows, or if there are no
// selections, the current row
func (m *Model[V]) SelectedOrCurrent() []V {
	if len(m.selected) > 0 {
		rows := make([]V, len(m.selected))
		var i int
		for _, v := range m.selected {
			rows[i] = v
			i++
		}
		return rows
	}
	if row, ok := m.CurrentRow(); ok {
		return []V{row}
	}
	return nil
}

// ToggleSelection toggles the selection of the current row.
func (m *Model[V]) ToggleSelection() {
	if !m.selectable {
		return
	}
	current, ok := m.CurrentRow()
	if !ok {
		return
	}
	if _, isSelected := m.selected[current.GetID()]; isSelected {
		delete(m.selected, current.GetID())
	} else {
		m.selected[current.GetID()] = current
	}
}

// ToggleSelectionByID toggles the selection of the row with the given ID. If
// the ID does not exist no action is taken.
func (m *Model[V]) ToggleSelectionByID(id resource.ID) {
	if !m.selectable {
		return
	}
	v, ok := m.items[id]
	if !ok {
		return
	}
	if _, isSelected := m.selected[id]; isSelected {
		delete(m.selected, id)
	} else {
		m.selected[id] = v
	}
}

// SelectAll selects all rows. Any rows not currently selected are selected.
func (m *Model[V]) SelectAll() {
	if !m.selectable {
		return
	}
	for _, row := range m.rows {
		m.selected[row.GetID()] = row
	}
}

// DeselectAll de-selects any rows that are currently selected
func (m *Model[V]) DeselectAll() {
	if !m.selectable {
		return
	}
	m.selected = make(map[resource.ID]V)
}

// SelectRange selects a range of rows. If the current row is *below* a selected
// row then rows between them are selected, including the current row.
// Otherwise, if the current row is *above* a selected row then rows between
// them are selected, including the current row. If there are no selected rows
// then no action is taken.
func (m *Model[V]) SelectRange() {
	if !m.selectable {
		return
	}
	if len(m.selected) == 0 {
		return
	}
	// Determine the first row to select, and the number of rows to select.
	first := -1
	n := 0
	for i, row := range m.rows {
		if i == m.currentRowIndex && first > -1 && first < m.currentRowIndex {
			// Select rows before and including current row
			n = m.currentRowIndex - first + 1
			break
		}
		if _, ok := m.selected[row.GetID()]; !ok {
			// Ignore unselected rows
			continue
		}
		if i > m.currentRowIndex {
			// Select rows including current row and all rows up to but not
			// including next selected row
			first = m.currentRowIndex
			n = i - m.currentRowIndex
			break
		}
		// Start selecting rows after this currently selected row.
		first = i + 1
	}
	for _, row := range m.rows[first : first+n] {
		m.selected[row.GetID()] = row
	}
}

// SetItems overwrites all existing items in the table with items.
func (m *Model[V]) SetItems(items ...V) {
	m.items = make(map[resource.ID]V)
	m.rendered = make(map[resource.ID]RenderedRow)
	m.AddItems(items...)
}

// AddItems idem potently adds items to the table,
// updating any items that exist on the table already.
func (m *Model[V]) AddItems(items ...V) {
	for _, item := range items {
		// Add/update item
		m.items[item.GetID()] = item
		// (Re-)render item's row.
		m.rendered[item.GetID()] = m.rowRenderer(item)
	}
	m.filterRows(maps.Values(m.items)...)
}

func (m *Model[V]) removeItem(item V) {
	delete(m.rendered, item.GetID())
	delete(m.items, item.GetID())
	delete(m.selected, item.GetID())
	for i, row := range m.rows {
		if row.GetID() == item.GetID() {
			// TODO: this might well produce a memory leak. See note:
			// https://go.dev/wiki/SliceTricks#delete-without-preserving-order
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			break
		}
	}
	if item.GetID() == m.currentRowID {
		// If item being removed is the current row the make the row above it
		// the new current row. (MoveUp also calls setStart, see below).
		m.MoveUp(1)
	} else {
		// Removing item may well affect index of first visible row, so
		// re-calculate just in case.
		m.setStart()
	}
}

// filterRows processes and filters items for display
func (m *Model[V]) filterRows(items ...V) {
	// Process items with filtering
	m.processFilteredItems(items)

	// Sort rows and locate current row
	m.sortAndLocateCurrentRow()

	// Set start index
	m.setStart()
}

// processFilteredItems handles filtering of items for display
func (m *Model[V]) processFilteredItems(items []V) {
	selected := make(map[resource.ID]V)
	m.rows = make([]V, 0, len(items))

	for _, item := range items {
		m.rows = append(m.rows, item)
		if m.selectable {
			if _, ok := m.selected[item.GetID()]; ok {
				selected[item.GetID()] = item
			}
		}
	}
	m.selected = selected
}

// sortAndLocateCurrentRow sorts the rows and tracks the current row
func (m *Model[V]) sortAndLocateCurrentRow() {
	// Sort rows in-place
	if m.sortFunc != nil {
		slices.SortFunc(m.rows, func(i, j V) int {
			return m.sortFunc(i, j)
		})
	}

	// Track current row index
	m.currentRowIndex = -1
	for i, row := range m.rows {
		if row.GetID() == m.currentRowID {
			m.currentRowIndex = i
			break
		}
	}

	// Set default current row if needed
	if len(m.rows) > 0 && m.currentRowIndex == -1 {
		m.currentRowIndex = 0
		m.currentRowID = m.rows[m.currentRowIndex].GetID()
	}
}

// MoveUp moves the current row up by any number of rows.
// It can not go above the first row.
func (m *Model[V]) MoveUp(n int) {
	m.moveCurrentRow(-n)
}

// MoveDown moves the current row down by any number of rows.
// It can not go below the last row.
func (m *Model[V]) MoveDown(n int) {
	m.moveCurrentRow(n)
}

func (m *Model[V]) moveCurrentRow(n int) {
	if len(m.rows) > 0 {
		m.currentRowIndex = clamp(m.currentRowIndex+n, 0, len(m.rows)-1)
		m.currentRowID = m.rows[m.currentRowIndex].GetID()
		m.setStart()
	}
}

func (m *Model[V]) setStart() {
	// Start index must be at least the current row index minus the max number
	// of visible rows.
	minimum := max(0, m.currentRowIndex-m.rowAreaHeight()+1)
	// Start index must be at most the lesser of:
	// (a) the current row index, or
	// (b) the number of rows minus the maximum number of visible rows (as many
	// rows as possible are rendered)
	maximum := max(0, min(m.currentRowIndex, len(m.rows)-m.rowAreaHeight()))
	m.start = clamp(m.start, minimum, maximum)
}

// GotoTop makes the top row the current row.
func (m *Model[V]) GotoID(id resource.ID) tea.Cmd {
	if id == m.currentRowID {
		return nil
	}
	if _, ok := m.items[id]; ok {
		m.currentRowID = id
		m.currentRowIndex = 0
		for i, r := range m.rows {
			if r.GetID() == id {
				m.currentRowIndex = i
				break
			}
		}
	}
	m.setStart()
	item := m.items[m.currentRowID]

	return tui.CmdHandler(RowSelectedActionMsg[V]{
		Row:   item,
		RowID: item.GetID(),
	})
}

// GotoTop makes the top row the current row.
func (m *Model[V]) GotoTop() {
	m.MoveUp(m.currentRowIndex)
}

// GotoBottom makes the bottom row the current row.
func (m *Model[V]) GotoBottom() {
	m.MoveDown(len(m.rows))
}

// GetNextRowIDRelativeToCurrentRow returns the ID of the row after the current row
// If the current row is the last row, it returns previous row ID
// If the current row is the last row, it returns 0
func (m *Model[V]) GetNextRowIDRelativeToCurrentRow() resource.ID {
	if m.currentRowIndex+1 >= len(m.rows) {
		// If the current row is the last row, return previous row ID
		if m.currentRowIndex-1 >= 0 {
			return m.rows[m.currentRowIndex-1].GetID()
		}
		// If the current row is the first row, return 0
		return resource.ID(0)
	}
	return m.rows[m.currentRowIndex+1].GetID()
}

// GetNextRowIDRelativeToCurrentSelection returns the ID of the next row relative to the current selection.
// If there are no selected rows, it returns the next row relative to the current row.
// If the current selection is the last row, it returns the previous row ID.
// If the current selection is the first row, it returns the next row ID.
// If there are no rows, it returns 0.
func (m *Model[V]) GetNextRowIDRelativeToCurrentSelection() resource.ID {
	if len(m.selected) == 0 {
		return m.GetNextRowIDRelativeToCurrentRow()
	}
	// Get the first selected row
	selectedRows := maps.Values(m.selected)
	if len(selectedRows) == 0 {
		return resource.ID(0)
	}
	firstSelectedRow := selectedRows[0]
	for i, row := range m.rows {
		if row.GetID() == firstSelectedRow.GetID() {
			if i+1 < len(m.rows) {
				return m.rows[i+1].GetID()
			}
			if i-1 >= 0 {
				return m.rows[i-1].GetID()
			}
			return resource.ID(0)
		}
	}
	return resource.ID(0)
}

func (m *Model[V]) headersView() string {
	s := make([]string, 0, len(m.cols))
	// TODO: use index and don't use append below
	for _, col := range m.cols {
		style := lipgloss.NewStyle().
			Width(col.Width).
			MaxWidth(col.Width).
			Inline(true).
			Bold(true).
			Italic(true)
		if col.RightAlign {
			style = style.AlignHorizontal(lipgloss.Right)
		}
		renderedCell := style.Render(TruncateRight(col.Title, col.Width, "…"))
		s = append(s, m.styles.Cell.Render(renderedCell))
	}
	return lipgloss.NewStyle().
		MaxWidth(m.width).
		Render(lipgloss.JoinHorizontal(lipgloss.Left, s...))
}

func (m *Model[V]) renderCells(
	row V, current, selected bool, rowEdited bool,
) []string {
	cells := m.rendered[row.GetID()]
	styledCells := make([]string, len(m.cols))
	for i, col := range m.cols {
		content := m.cellRenderer(row, cells[col.Key], i, rowEdited)

		// Truncate content if it is wider than column
		truncated := col.TruncationFunc(content, col.Width, "…")
		// Ensure content is all on one line.
		style := lipgloss.NewStyle().
			Width(col.Width).
			MaxWidth(col.Width).
			Inline(true)
		if col.RightAlign {
			style = style.AlignHorizontal(lipgloss.Right)
		}
		if current || selected {
			truncated = utils.RemoveAnsiCodes(truncated)
		}
		// For normal rows, just apply the regular styling
		inlined := style.Render(truncated)
		// Apply block-styling to content
		boxed := lipgloss.NewStyle().
			PaddingRight(1 + m.styles.Cell.GetPaddingLeft()).
			Render(inlined)
		styledCells[i] = boxed
	}
	return styledCells
}

func (m *Model[V]) renderRow(rowIdx int) string {
	row := m.rows[rowIdx]

	var (
		current  bool
		selected bool
	)
	if _, ok := m.selected[row.GetID()]; ok {
		selected = true
	}
	current = rowIdx == m.currentRowIndex
	rowStyle := *m.styles.Cell

	switch {
	case current && selected:
		rowStyle = *m.styles.CurrentAndSelectedRow
	case current:
		rowStyle = *m.styles.CurrentRow
	case selected:
		rowStyle = *m.styles.SelectedRow
	}

	rowEdited := false
	editor := m.editorsCache.Get(row.GetID())
	if current && editor != nil && editor.EditionInProgress() {
		rowStyle = rowStyle.Italic(true)
		rowEdited = true
	}
	renderedCells := m.renderCells(row, current, selected, rowEdited)

	// Join cells together to form a row, ensuring it doesn't exceed maximum
	// table width
	renderedRow := lipgloss.JoinHorizontal(lipgloss.Left, renderedCells...)
	// Apply row style
	renderedRow = rowStyle.
		MaxWidth(m.width).
		Render(renderedRow)

	return renderedRow
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}
