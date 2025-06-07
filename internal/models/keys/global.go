package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

type GlobalKeyMap struct {
	Search *key.Binding
	Quit   *key.Binding
	Help   *key.Binding
	Debug  *key.Binding
}

func GetGlobalKeyMap() *GlobalKeyMap {
	search := key.NewBinding(
		key.WithKeys("ctrl+f", "f3"),
		key.WithHelp("Ctrl+f/F3", "search"),
	)
	quit := key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("␛/Ctrl+c", "exit"),
	)
	help := key.NewBinding(
		key.WithKeys("h", "H", "f1", "alt+?", "alt+,"),
		key.WithHelp("F1/h/Alt+?", "close help"),
	)
	debug := key.NewBinding(
		key.WithKeys("f10", "f12"),
		key.WithHelp("F10/F12", "show debug info"),
	)

	return &GlobalKeyMap{
		Search: &search,
		Quit:   &quit,
		Help:   &help,
		Debug:  &debug,
	}
}
