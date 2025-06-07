package tabs

import (
	"github.com/fchastanet/shell-command-bookmarker/internal/services"
	"github.com/fchastanet/shell-command-bookmarker/internal/services/models"
	pkgTabs "github.com/fchastanet/shell-command-bookmarker/pkg/components/tabs"
)

const (
	// AvailableCommands represents commands that are available for use
	AvailableCommands pkgTabs.CategoryType = iota
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
}

// NewCategoryAdapter creates a new adapter for category conversions
func NewCategoryAdapter(historyService *services.HistoryService) *CategoryAdapter {
	return &CategoryAdapter{
		historyService: historyService,
	}
}

func (ca *CategoryAdapter) GetCategoryTabs() []pkgTabs.CategoryTab[models.CommandStatus] {
	return []pkgTabs.CategoryTab[models.CommandStatus]{
		{
			Title:       "Available",
			Type:        AvailableCommands,
			Count:       0,
			FilterState: pkgTabs.FilterState{FilterValue: ""},
			CommandTypes: []models.CommandStatus{
				models.CommandStatusSaved,
				models.CommandStatusImported,
			},
		},
		{
			Title:        "Saved",
			Type:         SavedCommands,
			Count:        0,
			FilterState:  pkgTabs.FilterState{FilterValue: ""},
			CommandTypes: []models.CommandStatus{models.CommandStatusSaved},
		},
		{
			Title:        "New",
			Type:         NewCommands,
			Count:        0,
			FilterState:  pkgTabs.FilterState{FilterValue: ""},
			CommandTypes: []models.CommandStatus{models.CommandStatusImported},
		},
		{
			Title:        "Deleted",
			Type:         DeletedCommands,
			Count:        0,
			FilterState:  pkgTabs.FilterState{FilterValue: ""},
			CommandTypes: []models.CommandStatus{models.CommandStatusDeleted},
		},
		{
			Title:       "All",
			Type:        AllCommands,
			Count:       0,
			FilterState: pkgTabs.FilterState{FilterValue: ""},
			CommandTypes: []models.CommandStatus{
				models.CommandStatusSaved,
				models.CommandStatusImported,
				models.CommandStatusDeleted,
				models.CommandStatusObsolete,
			},
		},
	}
}

// GetCommandTypesByCategory returns command statuses for a UI category type
func (ca *CategoryAdapter) GetCategoryTabConfiguration(
	category pkgTabs.CategoryType,
) pkgTabs.CategoryTab[models.CommandStatus] {
	return ca.GetCategoryTabs()[category]
}

// GetCategoryCounts maps service-level category counts to UI category types
func (ca *CategoryAdapter) GetCategoryCounts() (map[pkgTabs.CategoryType]int, error) {
	// Get category counts from service
	serviceCounts, err := ca.historyService.GetCommandCountsByCategory()
	if err != nil {
		return nil, err
	}

	// Map service categories to UI categories
	uiCounts := make(map[pkgTabs.CategoryType]int)
	uiCounts[AvailableCommands] = serviceCounts[services.CommandCategoryAvailable]
	uiCounts[SavedCommands] = serviceCounts[services.CommandCategorySaved]
	uiCounts[NewCommands] = serviceCounts[services.CommandCategoryNew]
	uiCounts[DeletedCommands] = serviceCounts[services.CommandCategoryDeleted]
	uiCounts[AllCommands] = serviceCounts[services.CommandCategoryAll]

	return uiCounts, nil
}
