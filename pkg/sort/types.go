package sort

import (
	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
	"github.com/fchastanet/shell-command-bookmarker/pkg/tui"
)

// Field represents a field that can be sorted
type Field string

// Direction represents the direction of sorting
type Direction string

const (
	// Sort directions
	DirectionAsc  Direction = "▲"
	DirectionDesc Direction = "▼"
)

// Option represents a sort option (field and direction)
type Option[FieldType string] struct {
	Field     FieldType
	Direction Direction
}

// State contains the current sort configuration
type State[ElementType resource.Identifiable, FieldType string] struct {
	EditorSortStyles   EditorSortStylesInterface
	IDField            FieldType
	PrimarySort        *Option[FieldType]
	SecondarySort      *Option[FieldType]
	CompareBySortField CompareBySortFieldFunc[ElementType, FieldType]
	KeyMap             *KeyMap
	Fields             []FieldType
	SelectedField      SelectedField // Currently selected field
	IsEditActive       bool          // Whether sort edit mode is active
}

type CompareBySortFieldFunc[ElementType resource.Identifiable, FieldType string] func(
	i, j ElementType, field FieldType,
) int

type SelectedField int

const (
	SelectedFieldPrimaryField       SelectedField = iota // Primary sort field
	SelectedFieldPrimaryDirection                        // Primary sort direction
	SelectedFieldSecondaryField                          // Secondary sort field
	SelectedFieldSecondaryDirection                      // Secondary sort direction
)

// Msg is sent when sorting is applied
type Msg[ElementType resource.Identifiable, FieldType string] struct {
	State   *State[ElementType, FieldType]
	InfoMsg *tui.InfoMsg
}

type MsgSortEditModeChanged[ElementType resource.Identifiable, FieldType string] struct {
	State *State[ElementType, FieldType]
}

// NewDefaultState creates a new default sort state
func NewDefaultState[ElementType resource.Identifiable, FieldType string](
	editorStyles EditorSortStylesInterface,
	idField FieldType,
	fields []FieldType,
	keyMap *KeyMap,
	compareBySortFieldFunc CompareBySortFieldFunc[ElementType, FieldType],
) *State[ElementType, FieldType] {
	return &State[ElementType, FieldType]{
		PrimarySort: &Option[FieldType]{
			Field:     idField,
			Direction: DirectionAsc,
		},
		SecondarySort:      nil,
		IsEditActive:       false,
		SelectedField:      0,
		EditorSortStyles:   editorStyles,
		Fields:             fields,
		IDField:            idField,
		CompareBySortField: compareBySortFieldFunc,
		KeyMap:             keyMap,
	}
}

// GetDirectionOptions returns the available sort direction options
func GetDirectionOptions() []Direction {
	return []Direction{
		DirectionAsc,
		DirectionDesc,
	}
}

func (state *State[ElementType, FieldType]) FixDuplicateSortKey() {
	if state.SecondarySort != nil && state.PrimarySort.Field == state.SecondarySort.Field {
		state.SecondarySort = nil
	}

	// Secondary selector (if primary is not ID)
	if state.PrimarySort.Field != state.IDField {
		// Initialize secondary sort if needed
		if state.SecondarySort == nil {
			state.SecondarySort = &Option[FieldType]{
				Field:     state.IDField,
				Direction: DirectionAsc,
			}
		}
	}
}
