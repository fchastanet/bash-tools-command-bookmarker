package keys

import (
	"github.com/charmbracelet/bubbles/key"
	pkgTabs "github.com/fchastanet/shell-command-bookmarker/pkg/components/tabs"
)

// FilterKeyMap is a key map of keys available in filter mode.
func GetFilterKeyMap() *pkgTabs.FilterKeyMap {
	nextTabKey := key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next tab"),
	)
	previousTabKey := key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "previous tab"),
	)
	filter := key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp(`/`, "filter"),
	)
	validate := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "exit filter"),
	)
	closeKey := key.NewBinding(
		key.WithKeys(
			"esc", "ctrl+c",
			"up", "down", "pgup", "pgdown", "ctrl+pgup", "ctrl+pgdown",
		),
		key.WithHelp("␛/Ctrl+c", "clear filter"),
	)
	return &pkgTabs.FilterKeyMap{
		Filter:      &filter,
		NextTab:     &nextTabKey,
		PreviousTab: &previousTabKey,
		Validate:    &validate,
		Close:       &closeKey,
	}
}
