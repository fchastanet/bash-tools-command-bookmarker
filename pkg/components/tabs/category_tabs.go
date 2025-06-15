package tabs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/pkg/category"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/sort"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

// CategoryTabStyles is an interface for styling category tabs
type CategoryTabStyles interface {
	GetActiveTabStyle() lipgloss.Style
	GetInactiveTabStyle() lipgloss.Style
	GetNavigationArrowStyle() lipgloss.Style
	GetTabCountStyle() lipgloss.Style
}

// CategoryTab represents a command category tab
type CategoryTab[ElementType resource.Identifiable, CommandStatus any, FieldType string] struct {
	Title        string
	FilterState  *category.FilterSortState[ElementType, FieldType]
	CommandTypes []CommandStatus
	Type         category.Type
	Count        int
}

// CategoryAdapterInterface defines methods for category tab operations
type CategoryAdapterInterface[
	ElementType resource.Identifiable,
	CommandStatus any,
	FieldType string,
] interface {
	// GetCategoryTabs returns the list of category tabs
	GetCategoryTabs(
		compareBySortFieldFunc sort.CompareBySortFieldFunc[ElementType, FieldType],
	) []CategoryTab[ElementType, CommandStatus, FieldType]

	// GetCategoryTabConfiguration returns the full category tab configuration
	GetCategoryTabConfiguration(
		category category.Type,
		compareBySortFieldFunc sort.CompareBySortFieldFunc[ElementType, FieldType],
	) CategoryTab[ElementType, CommandStatus, FieldType]
	// GetCategoryCounts returns the counts of commands in each category
	GetCategoryCounts() (map[category.Type]int, error)
}

type FilterKeyMap struct {
	Filter      *key.Binding
	NextTab     *key.Binding
	PreviousTab *key.Binding
	Validate    *key.Binding
	Close       *key.Binding
}

// CategoryTabs is the component that manages the navigation between different command categories
type CategoryTabs[
	ElementType resource.Identifiable,
	CommandStatus any,
	FieldType string,
] struct {
	styles        CategoryTabStylesInterface
	inputModel    InputModel
	adapter       CategoryAdapterInterface[ElementType, CommandStatus, FieldType] // Adapter for category-specific logic
	filterKeyMap  *FilterKeyMap
	tabs          []CategoryTab[ElementType, CommandStatus, FieldType]
	activeTabIdx  int
	width         int
	filteredCount int // Count of filtered items, if applicable
	focused       bool
}

// Message types for CategoryTabs events
type CategoryTabChangedMsg[
	ElementType resource.Identifiable,
	CommandStatus any,
	FieldType string,
] struct {
	NewTab *CategoryTab[ElementType, CommandStatus, FieldType]
}

// ErrCategoryTabNotFound is returned when no commands are selected for an operation
type ErrCategoryTabNotFound struct {
	tab category.Type
}

func (e *ErrCategoryTabNotFound) Error() string {
	return fmt.Sprintf("no category tab found for type %d", e.tab)
}

// InputModel is an interface that represents any filtering component
type InputModel interface {
	GetFilterValue() string
	SetFilterValue(string)
	Focus() tea.Cmd
	Blur()
	Focused() bool
	SetWidth(int)
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
}

const halfWidth = 2 // Used to divide the width for filter input

// NewCategoryTabs creates a new CategoryTabs component
func NewCategoryTabs[
	ElementType resource.Identifiable,
	CommandStatus any,
	FieldType string,
](
	styles CategoryTabStylesInterface,
	inputModel InputModel,
	adapter CategoryAdapterInterface[ElementType, CommandStatus, FieldType],
	filterKeyMap *FilterKeyMap,
	compareBySortFieldFunc sort.CompareBySortFieldFunc[ElementType, FieldType],
) *CategoryTabs[ElementType, CommandStatus, FieldType] {
	tabs := adapter.GetCategoryTabs(
		compareBySortFieldFunc,
	)

	return &CategoryTabs[ElementType, CommandStatus, FieldType]{
		styles:        styles,
		tabs:          tabs,
		activeTabIdx:  0,
		width:         0,
		inputModel:    inputModel,
		focused:       false,
		adapter:       adapter,
		filterKeyMap:  filterKeyMap,
		filteredCount: 0,
	}
}

// Init initializes the CategoryTabs component (implementation of tea.Model interface)
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) Init() tea.Cmd {
	return func() tea.Msg {
		return CategoryTabChangedMsg[ElementType, CommandStatus, FieldType]{
			NewTab: &ct.tabs[ct.activeTabIdx],
		}
	}
}

// Update handles messages and events
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ct.width = msg.Width
		ct.inputModel.SetWidth(msg.Width / halfWidth) // Half the width for the filter
	case tea.KeyMsg:
		if !ct.focused {
			break
		}
		return ct.handleKeyMsg(msg)
	case tea.FocusMsg:
		ct.focused = true
	case tea.BlurMsg:
		ct.focused = false
	case table.BulkInsertMsg[ElementType]:
		ct.filteredCount = len(msg.Items)
	}

	// Update filter model
	cmds = append(cmds, ct.inputModel.Update(msg))

	return tea.Batch(cmds...)
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) ChangeCategoryTab(
	newTabType category.Type,
) tea.Cmd {
	// Find the index of the new tab
	for i, tab := range ct.tabs {
		if tab.Type == newTabType {
			ct.activeTabIdx = i
			// Set the filter value for the new tab
			ct.inputModel.SetFilterValue(tab.FilterState.FilterValue)
			// Save the filter value in the tab's filter state
			ct.tabs[ct.activeTabIdx].FilterState.FilterValue = tab.FilterState.FilterValue
			// Return a command to notify about the tab change
			return func() tea.Msg {
				return CategoryTabChangedMsg[ElementType, CommandStatus, FieldType]{
					NewTab: &ct.tabs[ct.activeTabIdx],
				}
			}
		}
	}
	// If the new tab is not found, return error command
	return func() tea.Msg {
		return tui.ErrorMsg(&ErrCategoryTabNotFound{tab: newTabType})
	}
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) handleKeyMsg(keyMsg tea.KeyMsg) tea.Cmd {
	keys := ct.filterKeyMap
	switch {
	case tui.CheckKey(keyMsg, keys.Filter):
		if !ct.inputModel.Focused() {
			return ct.inputModel.Focus()
		}
	case tui.CheckKey(keyMsg, keys.PreviousTab):
		// Switch to the previous category tab
		return ct.prevCategory()
	case tui.CheckKey(keyMsg, keys.NextTab):
		// Switch to the next category tab
		return ct.nextCategory()
	case tui.CheckKey(keyMsg, keys.Validate):
		if ct.inputModel.Focused() {
			ct.inputModel.Blur()
			return ct.handleValidate()
		}
	case tui.CheckKey(keyMsg, keys.Close):
		if ct.inputModel.Focused() {
			ct.inputModel.Blur()
			return tui.GetDummyCmd()
		}
	}

	return ct.handleFilterInput(keyMsg)
}

// handleFilterInput handles passing keystrokes to the filter when it's focused
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) handleFilterInput(keyMsg tea.KeyMsg) tea.Cmd {
	// If the filter is visible, pass the key message to the filter model
	if ct.inputModel.Focused() {
		cmd := ct.inputModel.Update(keyMsg)
		return tea.Batch(cmd, ct.handleValidate())
	}
	return nil
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) handleValidate() tea.Cmd {
	filterValue := ct.inputModel.GetFilterValue()
	ct.tabs[ct.activeTabIdx].FilterState.FilterValue = filterValue
	return func() tea.Msg {
		return CategoryTabChangedMsg[ElementType, CommandStatus, FieldType]{
			NewTab: &ct.tabs[ct.activeTabIdx],
		}
	}
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) FilterActive() bool {
	return ct.inputModel.Focused()
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) GetActiveTab() *CategoryTab[ElementType, CommandStatus, FieldType] {
	return &ct.tabs[ct.activeTabIdx]
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) GetActiveTabTitle() string {
	return ct.tabs[ct.activeTabIdx].Title
}

// prevCategory selects the previous category tab
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) prevCategory() tea.Cmd {
	prevTabIdx := ct.activeTabIdx

	if ct.activeTabIdx == 0 {
		ct.activeTabIdx = len(ct.tabs) - 1
	} else {
		ct.activeTabIdx--
	}

	return ct.categoryChangedMsg(prevTabIdx)
}

// nextCategory selects the next category tab
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) nextCategory() tea.Cmd {
	prevTabIdx := ct.activeTabIdx

	if ct.activeTabIdx == len(ct.tabs)-1 {
		ct.activeTabIdx = 0
	} else {
		ct.activeTabIdx++
	}

	return ct.categoryChangedMsg(prevTabIdx)
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) categoryChangedMsg(
	prevTabIdx int,
) tea.Cmd {
	// Save current filter value
	ct.tabs[prevTabIdx].FilterState.FilterValue = ct.inputModel.GetFilterValue()

	// Restore the filter value for the newly selected tab
	ct.inputModel.SetFilterValue(ct.tabs[ct.activeTabIdx].FilterState.FilterValue)

	return func() tea.Msg {
		return CategoryTabChangedMsg[ElementType, CommandStatus, FieldType]{
			NewTab: &ct.tabs[ct.activeTabIdx],
		}
	}
}

// GetActiveCategory returns the currently active category
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) GetActiveCategory() category.Type {
	return ct.tabs[ct.activeTabIdx].Type
}

func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) GetActiveFilter() string {
	return ct.tabs[ct.activeTabIdx].FilterState.FilterValue
}

// GetActiveSortState returns the currently active sort state
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) GetActiveSortState() *sort.State[ElementType, FieldType] {
	return ct.tabs[ct.activeTabIdx].FilterState.SortState
}

// SetActiveSortState sets the sort state for the active tab
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) SetActiveSortState(
	state *sort.State[ElementType, FieldType],
) {
	if ct.activeTabIdx >= 0 && ct.activeTabIdx < len(ct.tabs) {
		ct.tabs[ct.activeTabIdx].FilterState.SortState = state
	}
}

// GetCommandTypes returns the command status types for the active category
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) GetActiveTabCommandTypes() []CommandStatus {
	return ct.tabs[ct.activeTabIdx].CommandTypes
}

// SetCounts updates the counts for each category
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) SetCounts(counts map[category.Type]int) {
	for i := range ct.tabs {
		if count, ok := counts[ct.tabs[i].Type]; ok {
			ct.tabs[i].Count = count
		}
	}
}

// UpdateCategoryCounts fetches and updates counts from the service
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) UpdateCategoryCounts() error {
	if ct.adapter == nil {
		return nil // No adapter, no updates
	}

	counts, err := ct.adapter.GetCategoryCounts()
	if err != nil {
		return err
	}

	ct.SetCounts(counts)
	return nil
}

// Focus gives focus to the component
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) Focus() tea.Cmd {
	ct.focused = true
	return nil
}

// Blur removes focus from the component
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) Blur() {
	ct.focused = false
	ct.inputModel.Blur()
}

// View renders the component
func (ct *CategoryTabs[ElementType, CommandStatus, FieldType]) View() string {
	if ct.width == 0 {
		return ""
	}

	var builder strings.Builder

	// Left arrow
	leftArrow := ct.styles.GetNavigationArrowStyle().Render("◀")

	// Tabs
	renderedTabs := make([]string, len(ct.tabs))
	for i, tab := range ct.tabs {
		if i == ct.activeTabIdx {
			count := ct.styles.GetTabCountStyle().Render(fmt.Sprintf("(%d)", tab.Count))
			renderedTabs[i] = ct.styles.GetActiveTabStyle().Render(tab.Title + " " + count)
		} else {
			count := ct.styles.GetTabCountStyle().Render(fmt.Sprintf("(%d)", tab.Count))
			renderedTabs[i] = ct.styles.GetInactiveTabStyle().Render(tab.Title + " " + count)
		}
	}

	// Right arrow
	rightArrow := ct.styles.GetNavigationArrowStyle().Render("▶")

	// Join all elements
	tabsContent := leftArrow + " " + strings.Join(renderedTabs, " ") + " " + rightArrow

	// Center the tabs
	paddedTabsContent := lipgloss.PlaceHorizontal(ct.width, lipgloss.Center, tabsContent)
	builder.WriteString(paddedTabsContent)
	builder.WriteString("\n")

	// Add a horizontal line separator - full width
	// Ensure we don't use a negative or zero width
	separatorWidth := ct.width
	if separatorWidth <= 0 {
		// If width is not set properly, use the width of the tabsContent as a fallback
		separatorWidth = lipgloss.Width(tabsContent)
	}

	// Make sure separatorWidth is positive to avoid panic
	if separatorWidth > 0 {
		builder.WriteString(strings.Repeat("─", separatorWidth))
	} else {
		// Fallback to a minimum separator
		builder.WriteString("────────")
	}
	builder.WriteString("\n")

	// In sort active mode, we show sort UI, otherwise, we show the regular filter UI
	activeSortState := ct.GetActiveSortState()
	sortView := ""
	if activeSortState != nil {
		// Render the sort UI
		sortView = activeSortState.View()
		sortView += "    "
	}

	filterView := ""
	if ct.inputModel.Focused() || ct.inputModel.GetFilterValue() != "" {
		filterView = ct.inputModel.View() + " Filter items count:" + strconv.Itoa(ct.filteredCount)
	}

	builder.WriteString(
		lipgloss.JoinHorizontal(lipgloss.Left, sortView, filterView),
	)

	return builder.String()
}
