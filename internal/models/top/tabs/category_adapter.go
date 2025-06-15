package tabs

import (
	"github.com/fchastanet/shell-command-bookmarker/internal/models/structure"
	"github.com/fchastanet/shell-command-bookmarker/internal/services"
	dbmodels "github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	"github.com/fchastanet/shell-command-bookmarker/pkg/category"
	pkgTabs "github.com/fchastanet/shell-command-bookmarker/pkg/components/tabs"
	"github.com/fchastanet/shell-command-bookmarker/pkg/sort"
)

const (
	// AvailableCommands represents commands that are available for use
	AvailableCommands category.Type = iota
	// SavedCommands represents commands that have been saved
	SavedCommands
	// NewCommands represents commands that have been imported but not yet saved
	NewCommands
	// DeletedCommands represents commands that have been marked as deleted
	DeletedCommands
	// AllCommands represents all commands regardless of status
	AllCommands
)

// CategoryAdapter helps translate between UI category types and service-level categories
type CategoryAdapter struct {
	historyService *services.HistoryService
	sortStyles     sort.EditorSortStylesInterface
	sortKeyMap     *sort.KeyMap
}

// NewCategoryAdapter creates a new adapter for category conversions
func NewCategoryAdapter(
	historyService *services.HistoryService,
	sortStyles sort.EditorSortStylesInterface,
	sortKeyMap *sort.KeyMap,
) *CategoryAdapter {
	return &CategoryAdapter{
		historyService: historyService,
		sortStyles:     sortStyles,
		sortKeyMap:     sortKeyMap,
	}
}

func (ca *CategoryAdapter) GetCategoryTabs(
	compareBySortFieldFunc sort.CompareBySortFieldFunc[*dbmodels.Command, string],
) []pkgTabs.CategoryTab[
	*dbmodels.Command,
	dbmodels.CommandStatus,
	string,
] {
	sortFields := []string{
		structure.FieldID,
		structure.FieldTitle,
		structure.FieldScript,
		structure.FieldStatus,
		structure.FieldLintStatus,
		structure.FieldCreationDate,
		structure.FieldModificationDate,
		structure.FieldFilterScore,
	}

	// Create a function that returns a new sort state for each tab
	createNewSortState := func() *sort.State[*dbmodels.Command, string] {
		sortState := sort.NewDefaultState(
			ca.sortStyles,
			structure.FieldFilterScore,
			sortFields,
			ca.sortKeyMap,
			compareBySortFieldFunc,
		)
		sortState.PrimarySort.Direction = sort.DirectionDesc
		sortState.SecondarySort = &sort.Option[structure.Field]{
			Field:     structure.FieldID,
			Direction: sort.DirectionAsc,
		}
		return sortState
	}

	return []pkgTabs.CategoryTab[
		*dbmodels.Command,
		dbmodels.CommandStatus,
		string,
	]{
		newCategoryTab(
			"Available",
			createNewSortState(),
			AvailableCommands,
			[]dbmodels.CommandStatus{
				dbmodels.CommandStatusSaved,
				dbmodels.CommandStatusImported,
			},
		),
		newCategoryTab(
			"Saved",
			createNewSortState(),
			SavedCommands,
			[]dbmodels.CommandStatus{
				dbmodels.CommandStatusSaved,
			},
		),
		newCategoryTab(
			"New",
			createNewSortState(),
			NewCommands,
			[]dbmodels.CommandStatus{
				dbmodels.CommandStatusImported,
			},
		),
		newCategoryTab(
			"Deleted",
			createNewSortState(),
			DeletedCommands,
			[]dbmodels.CommandStatus{
				dbmodels.CommandStatusDeleted,
			},
		),
		newCategoryTab(
			"All",
			createNewSortState(),
			AllCommands,
			[]dbmodels.CommandStatus{
				dbmodels.CommandStatusSaved,
				dbmodels.CommandStatusImported,
				dbmodels.CommandStatusDeleted,
				dbmodels.CommandStatusObsolete,
			},
		),
	}
}

func newCategoryTab(
	title string,
	sortState *sort.State[*dbmodels.Command, string],
	categoryType category.Type,
	commandTypes []dbmodels.CommandStatus,
) pkgTabs.CategoryTab[
	*dbmodels.Command,
	dbmodels.CommandStatus,
	string,
] {
	return pkgTabs.CategoryTab[
		*dbmodels.Command,
		dbmodels.CommandStatus,
		string,
	]{
		Title: title,
		Type:  categoryType,
		Count: 0,
		FilterState: &category.FilterSortState[*dbmodels.Command, string]{
			FilterValue: "",
			SortState:   sortState,
		},
		CommandTypes: commandTypes,
	}
}

// GetCommandTypesByCategory returns command statuses for a UI category type
func (ca *CategoryAdapter) GetCategoryTabConfiguration(
	cat category.Type,
	compareBySortFieldFunc sort.CompareBySortFieldFunc[*dbmodels.Command, string],
) pkgTabs.CategoryTab[*dbmodels.Command, dbmodels.CommandStatus, string] {
	return ca.GetCategoryTabs(compareBySortFieldFunc)[cat]
}

// GetCategoryCounts maps service-level category counts to UI category types
func (ca *CategoryAdapter) GetCategoryCounts() (map[category.Type]int, error) {
	// Get category counts from service
	serviceCounts, err := ca.historyService.GetCommandCountsByCategory()
	if err != nil {
		return nil, err
	}

	// Map service categories to UI categories
	uiCounts := make(map[category.Type]int)
	uiCounts[AvailableCommands] = serviceCounts[services.CommandCategoryAvailable]
	uiCounts[SavedCommands] = serviceCounts[services.CommandCategorySaved]
	uiCounts[NewCommands] = serviceCounts[services.CommandCategoryNew]
	uiCounts[DeletedCommands] = serviceCounts[services.CommandCategoryDeleted]
	uiCounts[AllCommands] = serviceCounts[services.CommandCategoryAll]

	return uiCounts, nil
}
