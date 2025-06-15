package structure

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/keys"
	"github.com/fchastanet/shell-command-bookmarker/pkg/components/tabs"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/sort"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

// alter how all messages are handled.
type Mode int

const (
	NormalMode Mode = iota // default
	PromptMode             // confirm prompt is visible and taking input
)

type ChangeModeMsg struct {
	NewMode Mode
}

type KeyMaps struct {
	Sort              *sort.KeyMap
	Filter            *tabs.FilterKeyMap
	Global            *keys.GlobalKeyMap
	Pane              *keys.PaneNavigationKeyMap
	TableNavigation   *table.Navigation
	TableAction       *table.Action
	TableCustomAction *keys.TableCustomActionKeyMap
	Editor            *keys.EditorKeyMap
	Form              *huh.KeyMap
}

type ChildModel interface {
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
	BeforeSwitchPane() tea.Cmd
}

// Page identifies an instance of a model
type Page struct {
	// The model kind. Identifies the model maker to construct the page.
	Kind resource.Kind
	// ID of resource for a model. If the model does not have a single resource
	// but is say a listing of resources, then this is nil.
	ID resource.ID
}

// ModelHelpBindings is implemented by models that surface further help bindings
// specific to the model.
type ModelHelpBindings interface {
	HelpBindings() []*key.Binding
}
