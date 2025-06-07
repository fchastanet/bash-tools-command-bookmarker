package tabs

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

type Tab struct {
	Model     tea.Model
	tabsModel *Tabs
	Title     string
}

type tabsSettings struct {
	Keys keyMap
}

type TabStyles interface {
	GetTabBorderStyle(
		isFirst bool,
		isLast bool,
		isActive bool,
		width int,
		tabsCount int,
	) lipgloss.Style
	GetActiveTabStyle() lipgloss.Style
	GetWindowStyle() lipgloss.Style
}

type Tabs struct {
	styles          TabStyles
	settings        *tabsSettings
	Tabs            []Tab
	width           int
	height          int
	activeTab       int
	terminalFocused bool
}

const OuterTabContentHeight = 8

func defaultKeyMap() keyMap {
	left := key.NewBinding(
		key.WithKeys("left", "p"),
		key.WithHelp("←/p", "move tab left"),
	)
	right := key.NewBinding(
		key.WithKeys("right", "n"),
		key.WithHelp("→/n", "move tab right"),
	)
	return keyMap{
		Left:  &left,
		Right: &right,
	}
}

func NewTabs(
	tabs []Tab,
	styles TabStyles,
) *Tabs {
	tabsModel := &Tabs{
		width:  0,
		height: 0,
		Tabs:   tabs,
		settings: &tabsSettings{
			Keys: defaultKeyMap(),
		},
		styles:          styles,
		activeTab:       0,
		terminalFocused: true,
	}

	// set tabs parent
	for i := range tabs {
		tabs[i].tabsModel = tabsModel
	}
	return tabsModel
}

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Left  *key.Binding
	Right *key.Binding
}

func (t *Tabs) GetKeyBindings() []*key.Binding {
	return []*key.Binding{
		t.settings.Keys.Left, t.settings.Keys.Right,
	}
}

func (t *Tabs) Init() tea.Cmd {
	batches := make([]tea.Cmd, len(t.Tabs))
	for _, tab := range t.Tabs {
		if tab.Model == nil {
			continue
		}
		batches = append(batches, tab.Model.Init())
	}
	return tea.Batch(batches...)
}

func (t *Tabs) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if t.activeTab != -1 {
		newM, cmd := t.Tabs[t.activeTab].Model.Update(msg)
		t.Tabs[t.activeTab].Model = newM
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
	case tea.FocusMsg:
		t.terminalFocused = true
	case tea.BlurMsg:
		t.terminalFocused = false

	case tea.KeyMsg:
		if !t.terminalFocused {
			return t, tea.Batch(cmds...)
		}
		t.updateActiveTab(msg)
	}
	return t, tea.Batch(cmds...)
}

func (t *Tabs) updateActiveTab(msg tea.KeyMsg) {
	oldActiveTab := t.activeTab
	keys := t.settings.Keys
	switch {
	case key.Matches(msg, *keys.Right) && keys.Right.Enabled():
		if t.activeTab == len(t.Tabs)-1 {
			t.activeTab = 0
		} else {
			t.activeTab = min(t.activeTab+1, len(t.Tabs)-1)
		}
	case key.Matches(msg, *keys.Left) && keys.Left.Enabled():
		if t.activeTab == 0 {
			t.activeTab = len(t.Tabs) - 1
		} else {
			t.activeTab = max(t.activeTab-1, 0)
		}
	}
	if oldActiveTab != t.activeTab {
		activeTab := t.Tabs[t.activeTab].Model
		tabFrameWidth := t.styles.GetActiveTabStyle().GetHorizontalFrameSize()
		tabFrameHeight := t.styles.GetActiveTabStyle().GetVerticalFrameSize()
		activeTab.Update(tea.FocusMsg{})
		activeTab.Update(tea.WindowSizeMsg{
			Width:  t.width - tabFrameWidth,
			Height: t.height - tabFrameHeight - OuterTabContentHeight,
		})
		t.Tabs[oldActiveTab].Model.Update(tea.BlurMsg{})
		// TODO t.focusManager.SetCurrentFocus(t.Tabs[t.activeTab].Model)
	}
}

func (tab *Tab) View(
	tabsCount int, width int,
	isFirst bool, isLast bool, isActive bool,
) string {
	if tab.Model == nil {
		return ""
	}
	borderStyle := tab.tabsModel.styles.GetTabBorderStyle(
		isFirst, isLast, isActive,
		width, tabsCount,
	)
	return borderStyle.Render(tab.Title)
}

func (t *Tabs) View() string {
	doc := strings.Builder{}

	renderedTabs := make([]string, len(t.Tabs))

	for i, tab := range t.Tabs {
		if tab.Model == nil {
			continue
		}
		tabsCount := len(t.Tabs)
		isFirst, isLast, isActive := i == 0, i == len(t.Tabs)-1, i == t.activeTab
		renderedTabs[i] = tab.View(
			tabsCount, t.width,
			isFirst, isLast, isActive,
		)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n")
	doc.WriteString(
		t.styles.GetWindowStyle().Width(
			lipgloss.Width(row) - t.styles.GetWindowStyle().GetHorizontalFrameSize(),
		).Render(t.Tabs[t.activeTab].Model.View()),
	)
	return doc.String()
}
