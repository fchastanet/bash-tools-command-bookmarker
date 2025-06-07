package tabs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

// CategoryType defines the type of command category
type CategoryType int

// CategoryTabStyles is an interface for styling category tabs
type CategoryTabStyles interface {
	GetActiveTabStyle() lipgloss.Style
	GetInactiveTabStyle() lipgloss.Style
	GetNavigationArrowStyle() lipgloss.Style
	GetTabCountStyle() lipgloss.Style
}

// FilterState holds the current filter value
type FilterState struct {
	FilterValue string
}

// CategoryTab represents a command category tab
type CategoryTab[CommandStatus any] struct {
	Title        string
	FilterState  FilterState
	CommandTypes []CommandStatus
	Type         CategoryType
	Count        int
}

type CategoryAdapterInterface[V resource.Identifiable, CommandStatus any] interface {
	// GetCategoryTabs returns the list of category tabs
	GetCategoryTabs() []CategoryTab[CommandStatus]
	// GetCategoryTabConfiguration returns the full category tab configuration
	GetCategoryTabConfiguration(category CategoryType) CategoryTab[CommandStatus]
	// GetCategoryCounts returns the counts of commands in each category
	GetCategoryCounts() (map[CategoryType]int, error)
}

type FilterKeyMap struct {
	Filter      *key.Binding
	NextTab     *key.Binding
	PreviousTab *key.Binding
	Validate    *key.Binding
	Close       *key.Binding
}

// CategoryTabs is the component that manages the navigation between different command categories
type CategoryTabs[V resource.Identifiable, CommandStatus any] struct {
	styles        CategoryTabStyles
	inputModel    InputModel
	adapter       CategoryAdapterInterface[V, CommandStatus] // Adapter for category-specific logic
	keyMaps       *FilterKeyMap
	tabs          []CategoryTab[CommandStatus]
	activeTabIdx  int
	width         int
	filteredCount int // Count of filtered items, if applicable
	focused       bool
}

// Message types for CategoryTabs events
type CategoryTabChangedMsg struct {
	Filter     string
	PrevTab    CategoryType
	CurrentTab CategoryType
}

type ChangeCategoryTabMsg struct {
	Filter string
	NewTab CategoryType
}

// ErrCategoryTabNotFound is returned when no commands are selected for an operation
type ErrCategoryTabNotFound struct {
	tab CategoryType
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
func NewCategoryTabs[V resource.Identifiable, CommandStatus any](
	styles CategoryTabStyles,
	inputModel InputModel,
	adapter CategoryAdapterInterface[V, CommandStatus],
	keyMaps *FilterKeyMap,
) *CategoryTabs[V, CommandStatus] {
	tabs := adapter.GetCategoryTabs()

	return &CategoryTabs[V, CommandStatus]{
		styles:        styles,
		tabs:          tabs,
		activeTabIdx:  0,
		width:         0,
		inputModel:    inputModel,
		focused:       false,
		adapter:       adapter,
		keyMaps:       keyMaps,
		filteredCount: 0,
	}
}

// Init initializes the CategoryTabs component (implementation of tea.Model interface)
func (ct *CategoryTabs[V, CommandStatus]) Init() tea.Cmd {
	return nil
}

// Update handles messages and events
func (ct *CategoryTabs[V, CommandStatus]) Update(msg tea.Msg) tea.Cmd {
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
	case table.BulkInsertMsg[V]:
		ct.filteredCount = len(msg.Items)
	case ChangeCategoryTabMsg:
		return ct.changeCategoryTab(msg.NewTab, msg.Filter)
	}

	// Update filter model
	cmds = append(cmds, ct.inputModel.Update(msg))

	return tea.Batch(cmds...)
}

func (ct *CategoryTabs[V, CommandStatus]) changeCategoryTab(newTab CategoryType, filter string) tea.Cmd {
	// Find the index of the new tab
	for i, tab := range ct.tabs {
		if tab.Type == newTab {
			ct.activeTabIdx = i
			// Set the filter value for the new tab
			ct.inputModel.SetFilterValue(filter)
			// Save the filter value in the tab's filter state
			ct.tabs[ct.activeTabIdx].FilterState.FilterValue = filter
			// Return a command to notify about the tab change
			return func() tea.Msg {
				return CategoryTabChangedMsg{
					PrevTab:    ct.tabs[ct.activeTabIdx].Type,
					CurrentTab: newTab,
					Filter:     filter,
				}
			}
		}
	}
	// If the new tab is not found, return error command
	return func() tea.Msg {
		return tui.ErrorMsg(&ErrCategoryTabNotFound{tab: newTab})
	}
}

//nolint:cyclop // not really complex
func (ct *CategoryTabs[V, CommandStatus]) handleKeyMsg(keyMsg tea.KeyMsg) tea.Cmd {
	keys := ct.keyMaps
	switch {
	case key.Matches(keyMsg, *keys.Filter) && keys.Filter.Enabled():
		if !ct.inputModel.Focused() {
			ct.inputModel.Focus()
		}
	case key.Matches(keyMsg, *keys.PreviousTab) && keys.PreviousTab.Enabled():
		// Switch to the previous category tab
		return ct.prevCategory()
	case key.Matches(keyMsg, *keys.NextTab) && keys.NextTab.Enabled():
		// Switch to the next category tab
		return ct.nextCategory()
	case key.Matches(keyMsg, *keys.Validate) && keys.Validate.Enabled():
		if ct.inputModel.Focused() {
			ct.inputModel.Blur()
			return ct.handleValidate()
		}
	case key.Matches(keyMsg, *keys.Close) && keys.Close.Enabled():
		if ct.inputModel.Focused() {
			ct.inputModel.Blur()
			return tui.GetDummyCmd()
		}
	default:
		// If the filter is visible, pass the key message to the filter model
		if ct.inputModel.Focused() {
			cmd := ct.inputModel.Update(keyMsg)
			return tea.Batch(cmd, ct.handleValidate())
		}
	}
	return nil
}

func (ct *CategoryTabs[V, CommandStatus]) handleValidate() tea.Cmd {
	activeTabType := ct.tabs[ct.activeTabIdx].Type
	filterValue := ct.inputModel.GetFilterValue()
	ct.tabs[ct.activeTabIdx].FilterState.FilterValue = filterValue
	return func() tea.Msg {
		return CategoryTabChangedMsg{
			PrevTab:    activeTabType,
			CurrentTab: activeTabType,
			Filter:     filterValue,
		}
	}
}

func (ct *CategoryTabs[V, CommandStatus]) FilterActive() bool {
	return ct.inputModel.Focused()
}

func (ct *CategoryTabs[V, CommandStatus]) GetActiveTabTitle() string {
	return ct.tabs[ct.activeTabIdx].Title
}

// prevCategory selects the previous category tab
func (ct *CategoryTabs[V, CommandStatus]) prevCategory() tea.Cmd {
	prevTabIdx := ct.activeTabIdx
	prevTabType := ct.tabs[prevTabIdx].Type

	if ct.activeTabIdx == 0 {
		ct.activeTabIdx = len(ct.tabs) - 1
	} else {
		ct.activeTabIdx--
	}

	// Save current filter value
	ct.tabs[prevTabIdx].FilterState.FilterValue = ct.inputModel.GetFilterValue()

	// Restore the filter value for the newly selected tab
	ct.inputModel.SetFilterValue(ct.tabs[ct.activeTabIdx].FilterState.FilterValue)

	currentTabType := ct.tabs[ct.activeTabIdx].Type

	return func() tea.Msg {
		return CategoryTabChangedMsg{
			PrevTab:    prevTabType,
			CurrentTab: currentTabType,
			Filter:     ct.inputModel.GetFilterValue(),
		}
	}
}

// nextCategory selects the next category tab
func (ct *CategoryTabs[V, CommandStatus]) nextCategory() tea.Cmd {
	prevTabIdx := ct.activeTabIdx
	prevTabType := ct.tabs[prevTabIdx].Type

	if ct.activeTabIdx == len(ct.tabs)-1 {
		ct.activeTabIdx = 0
	} else {
		ct.activeTabIdx++
	}

	// Save current filter value
	ct.tabs[prevTabIdx].FilterState.FilterValue = ct.inputModel.GetFilterValue()

	// Restore the filter value for the newly selected tab
	ct.inputModel.SetFilterValue(ct.tabs[ct.activeTabIdx].FilterState.FilterValue)

	currentTabType := ct.tabs[ct.activeTabIdx].Type

	return func() tea.Msg {
		return CategoryTabChangedMsg{
			PrevTab:    prevTabType,
			CurrentTab: currentTabType,
			Filter:     ct.inputModel.GetFilterValue(),
		}
	}
}

// GetActiveCategory returns the currently active category
func (ct *CategoryTabs[V, CommandStatus]) GetActiveCategory() CategoryType {
	return ct.tabs[ct.activeTabIdx].Type
}

func (ct *CategoryTabs[V, CommandStatus]) GetActiveFilter() string {
	return ct.tabs[ct.activeTabIdx].FilterState.FilterValue
}

// GetCommandTypes returns the command status types for the active category
func (ct *CategoryTabs[V, CommandStatus]) GetCommandTypes() []CommandStatus {
	return ct.adapter.GetCategoryTabConfiguration(ct.tabs[ct.activeTabIdx].Type).CommandTypes
}

// SetCounts updates the counts for each category
func (ct *CategoryTabs[V, CommandStatus]) SetCounts(counts map[CategoryType]int) {
	for i := range ct.tabs {
		if count, ok := counts[ct.tabs[i].Type]; ok {
			ct.tabs[i].Count = count
		}
	}
}

// UpdateCategoryCounts fetches and updates counts from the service
func (ct *CategoryTabs[V, CommandStatus]) UpdateCategoryCounts() error {
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
func (ct *CategoryTabs[V, CommandStatus]) Focus() tea.Cmd {
	ct.focused = true
	return nil
}

// Blur removes focus from the component
func (ct *CategoryTabs[V, CommandStatus]) Blur() {
	ct.focused = false
	ct.inputModel.Blur()
}

// View renders the component
func (ct *CategoryTabs[V, CommandStatus]) View() string {
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

	if ct.inputModel.Focused() || ct.inputModel.GetFilterValue() != "" {
		filterView := ct.inputModel.View() + " Filter items count:" + strconv.Itoa(ct.filteredCount)
		builder.WriteString(filterView)
	}

	return builder.String()
}
