package top

import (
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/fchastanet/shell-command-bookmarker/internal/models"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/command"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/structure"
	"github.com/fchastanet/shell-command-bookmarker/internal/models/styles"
	"github.com/fchastanet/shell-command-bookmarker/internal/services"
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui/table"
)

// Maker makes new models
type Maker interface {
	Make(id resource.ID, width, height int) (structure.ChildModel, error)
}

// NewMakerFactory makes model makers for making models
func NewMakerFactory(
	editorsCache table.EditorsCacheInterface,
	app services.AppServiceInterface,
	myStyles *styles.Styles,
	spinnerObj *spinner.Model,
	keyMaps *KeyMaps,
) func(kind resource.Kind) models.Maker {
	makers := make(map[resource.Kind]models.Maker)
	makers[structure.CommandListKind] = &command.ListMaker{
		App:                     app.Self(),
		EditorsCache:            editorsCache,
		Styles:                  myStyles,
		Spinner:                 spinnerObj,
		TableCustomActionKeyMap: keyMaps.tableCustomAction,
		NavigationKeyMap:        keyMaps.tableNavigation,
		ActionKeyMap:            keyMaps.tableAction,
		FilterKeyMap:            keyMaps.filter,
	}
	makers[structure.SearchKind] = &command.SearchMaker{
		App:     app.Self(),
		Styles:  myStyles,
		Spinner: spinnerObj,
	}
	makers[structure.CommandEditorKind] = &command.EditorMaker{
		App:          app.Self(),
		Styles:       myStyles,
		EditorKeyMap: keyMaps.editor,
	}
	return func(kind resource.Kind) models.Maker {
		maker, ok := makers[kind]
		if !ok {
			return nil
		}
		return maker
	}
}
