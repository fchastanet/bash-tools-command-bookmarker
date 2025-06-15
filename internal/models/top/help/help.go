package help

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/keys"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/structure"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/styles"
	"github.com/fchastanet/shell-command-bookmarker/internal/services"
	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/sort"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

// BindingSet represents a group of key bindings with a title
type BindingSet struct {
	Title    string
	Bindings []*key.Binding
}

type PaneManagerHelpBindings interface {
	HelpBindings() []*key.Binding
}

// Model represents the help component
type Model struct {
	appService              services.AppServiceInterface
	paneManagerHelpBindings PaneManagerHelpBindings
	helpStyle               *lipgloss.Style
	styles                  *styles.Styles
	keyMaps                 *structure.KeyMaps
	selectedCommand         *dbmodels.Command
	currentSortState        *sort.State[*dbmodels.Command, string]
	bindingSets             []BindingSet
	width                   int
	height                  int
	mode                    structure.Mode
	focusedPane             structure.Position
	showHelp                bool
}

// New creates a new help component
func New(
	myStyles *styles.Styles,
	keyMaps *structure.KeyMaps,
	appService services.AppServiceInterface,
	paneManagerHelpBindings PaneManagerHelpBindings,
) Model {
	return Model{
		width:                   0,
		height:                  0,
		styles:                  myStyles,
		showHelp:                false,
		bindingSets:             []BindingSet{},
		helpStyle:               myStyles.HelpStyle.Main,
		selectedCommand:         nil,
		paneManagerHelpBindings: paneManagerHelpBindings,
		currentSortState:        nil,
		keyMaps:                 keyMaps,
		appService:              appService,
		focusedPane:             structure.TopPane,
		mode:                    structure.NormalMode,
	}
}

func (*Model) Init() tea.Cmd {
	// No initialization needed for help component
	return nil
}

// Update processes messages for the help component
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case structure.ChangeModeMsg:
		m.mode = msg.NewMode
		return m.updateHelpBindings()
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case structure.FocusedPaneChangedMsg:
		m.focusedPane = msg.To
		return m.updateHelpBindings()
	case tui.YesNoPromptMsg:
		return m.updateHelpBindings()
	case table.RowSelectedActionMsg[*dbmodels.Command]:
		return m.handleSelectedCommand(msg)
	case sort.Msg[*dbmodels.Command, string]:
		m.currentSortState = msg.State
		return m.updateHelpBindings()
	case sort.MsgSortEditModeChanged[*dbmodels.Command, string]:
		m.currentSortState = msg.State
		return m.updateHelpBindings()
	case tea.KeyMsg:
		if tui.CheckKey(msg, m.keyMaps.Global.Help) {
			m.showHelp = !m.showHelp
			return m.updateHelpBindings()
		}
	}

	// No updates needed for help component
	return nil
}

// handleWindowSize processes window size messages
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) tea.Cmd {
	m.width = msg.Width
	return nil
}

func (m *Model) handleSelectedCommand(
	msg table.RowSelectedActionMsg[*dbmodels.Command],
) tea.Cmd {
	m.selectedCommand = msg.Row
	return m.updateHelpBindings()
}

// updateHelpBindings updates the key bindings displayed in help
// based on current mode
func (m *Model) updateHelpBindings() tea.Cmd {
	// Clear previous binding sets
	m.clearBindingSets()
	m.updateHelpBindingsPromptMode()
	m.updateHelpBindingsNormalMode()
	return m.updateHelpHeight()
}

func (m *Model) updateHelpHeight() tea.Cmd {
	oldHeight := m.height
	m.height = m.maxBindingHeight()
	if oldHeight != m.height {
		return tui.GetResizeCmd(m, m.width, m.height)
	}
	return nil
}

func (m *Model) updateHelpBindingsPromptMode() {
	if m.mode != structure.PromptMode {
		return
	}
	// For prompt mode, just use a single set
	m.AddBindingSet("Prompt Controls", keys.GetFormBindings())
}

func (m *Model) updateHelpBindingsNormalMode() {
	if m.mode != structure.NormalMode {
		return
	}
	// For normal mode, organize bindings into logical groups
	if m.currentSortState != nil {
		sort.UpdateBindings(m.currentSortState.KeyMap, m.currentSortState)
		if m.currentSortState.IsEditActive {
			m.AddBindingSet("Sort Controls", keys.KeyMapToSlice(*m.currentSortState.KeyMap))
		}
	}
	m.AddBindingSet("Global", keys.KeyMapToSlice(*m.keyMaps.Global))
	m.AddBindingSet("Pane Navigation", m.paneManagerHelpBindings.HelpBindings())
	if m.focusedPane == structure.TopPane &&
		m.currentSortState != nil &&
		!m.currentSortState.IsEditActive {
		m.AddBindingSet("Filter Controls", keys.KeyMapToSlice(*m.keyMaps.Filter))
		m.AddBindingSet("Table Nav", keys.KeyMapToSlice(*m.keyMaps.TableNavigation))
		keys.UpdateBindings(
			m.keyMaps.TableAction,
			m.keyMaps.TableCustomAction,
			m.appService.IsShellSelectionMode(),
			m.selectedCommand,
		)

		tableCustomActions := keys.KeyMapToSlice(*m.keyMaps.TableCustomAction)
		tableCustomActions = append(tableCustomActions, m.currentSortState.KeyMap.Sort)

		m.AddBindingSet("Table Actions", keys.KeyMapToSlice(*m.keyMaps.TableAction))
		m.AddBindingSet("Command Actions", tableCustomActions)
	}
}

// IsVisible returns whether the help is currently visible
func (m *Model) IsVisible() bool {
	return m.showHelp
}

// Height returns the height of the help component when rendered
func (m *Model) Height() int {
	if m.showHelp {
		return m.maxBindingHeight() + m.styles.HelpStyle.BordersWidth
	}
	return 0
}

func (m *Model) maxBindingHeight() int {
	maxHeight := 0
	for _, set := range m.bindingSets {
		maxHeight = max(maxHeight, len(set.Bindings)+1) // +1 for the headers
	}
	return maxHeight
}

// Width returns the width of the help component
func (m *Model) Width() int {
	return m.width
}

// SetWidth updates the width of the help component
func (m *Model) SetWidth(width int) {
	m.width = width
}

// AddBindingSet adds a set of bindings with a title
func (m *Model) AddBindingSet(title string, bindings []*key.Binding) {
	m.bindingSets = append(m.bindingSets, BindingSet{
		Title:    title,
		Bindings: bindings,
	})
}

// clearBindingSets removes all binding sets
func (m *Model) clearBindingSets() {
	m.bindingSets = []BindingSet{}
}

// GetHelpWidget returns the help widget text
func (m *Model) GetHelpWidget() string {
	return m.helpStyle.Render("? help")
}

// View renders the help component
func (m *Model) View() string {
	if !m.showHelp {
		return ""
	}

	return m.viewBindingSets(m.maxBindingHeight())
}

// viewBindingSets renders the help component with binding sets
func (m *Model) viewBindingSets(rows int) string {
	columns := make([]string, 0, len(m.bindingSets))
	totalWidth := 0

	// Process each binding set
	for _, set := range m.bindingSets {
		bindings := removeDuplicateAndDisabledBindings(set.Bindings)
		if len(bindings) == 0 {
			continue
		}

		// Render the title with explicit left alignment
		title := m.styles.HelpStyle.TitleStyle.
			Render(set.Title)

		var helpKeys []string
		var descriptions []string

		// Add the title as the first row
		helpKeys = append(helpKeys, title)
		descriptions = append(descriptions, "")

		// Add the bindings
		for j := 0; j < min(rows-1, len(bindings)); j++ {
			helpKeys = append(helpKeys, m.styles.HelpStyle.KeyStyle.Render(bindings[j].Help().Key))
			descriptions = append(descriptions, m.styles.HelpStyle.DescStyle.Render(bindings[j].Help().Desc))
		}

		// Create columns for this set
		cols := []string{
			strings.Join(helpKeys, "\n"),
			strings.Join(descriptions, "\n"),
		}

		// Add spacing between sets
		if len(columns) > 0 {
			columns = append(columns, "   ")
		}

		// Join the columns horizontally
		setContent := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

		// Check if adding this set would exceed available width
		newWidth := totalWidth + lipgloss.Width(setContent) + m.styles.HelpStyle.ColumnMargin
		if len(columns) > 0 && newWidth > m.width-m.styles.HelpStyle.BordersWidth {
			break
		}

		totalWidth = newWidth
		columns = append(columns, setContent)
	}

	// Join all sets horizontally and enclose in a border
	content := lipgloss.JoinHorizontal(lipgloss.Top, columns...)
	return m.styles.PaneStyle.TopBorder.
		Height(rows).
		Width(m.width - m.styles.PaneStyle.BordersWidth).
		Render(content)
}

// removeDuplicateAndDisabledBindings removes duplicate bindings from a list of bindings. A
// binding is deemed a duplicate if another binding has the same list of keys.
func removeDuplicateAndDisabledBindings(bindings []*key.Binding) []*key.Binding {
	seen := make(map[string]struct{})
	var i int
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		bKey := strings.Join(b.Keys(), " ")
		if _, ok := seen[bKey]; ok {
			// duplicate, skip
			continue
		}
		seen[bKey] = struct{}{}
		bindings[i] = b
		i++
	}
	return bindings[:i]
}
