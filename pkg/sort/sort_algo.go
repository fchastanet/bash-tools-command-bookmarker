package sort

import (
	"time"

	"github.com/fchastanet/shell-command-bookmarker/pkg/resource"
)

// CommandSortFuncDynamic returns a sort function that dynamically gets the sort state
// from a getter function whenever sorting is performed. This ensures the sort always
// uses the current sort state rather than a cached state.
func CommandSortFuncDynamic[ElementType resource.Identifiable, FieldType string](
	getState func() *State[ElementType, FieldType],
) func(i, j ElementType) int {
	return func(i, j ElementType) int {
		// Get the current state whenever sorting is performed
		state := getState()
		if state == nil {
			// Fall back to default sorting if no state is available
			return CompareID(i, j)
		}

		// Primary sort
		primary := state.CompareBySortField(i, j, state.PrimarySort.Field)
		if primary != 0 {
			// Apply sort direction
			if state.PrimarySort.Direction == DirectionDesc {
				return -primary
			}
			return primary
		}

		// If primary fields are equal and secondary sort is defined
		if state.SecondarySort != nil {
			secondary := state.CompareBySortField(i, j, state.SecondarySort.Field)
			if secondary != 0 {
				// Apply sort direction
				if state.SecondarySort.Direction == DirectionDesc {
					return -secondary
				}
				return secondary
			}
		}

		// Fall back to ID comparison if everything else is equal
		return CompareID(i, j)
	}
}

func CompareInt(i, j int) int {
	if i < j {
		return -1
	} else if i > j {
		return 1
	}
	return 0
}

// compareTime compares two time values
func CompareTime(t1, t2 time.Time) int {
	switch {
	case t1.Before(t2):
		return -1
	case t1.After(t2):
		return 1
	default:
		return 0
	}
}

func CompareID[ElementType resource.Identifiable](i, j ElementType) int {
	switch {
	case i.GetID() < j.GetID():
		return -1
	case i.GetID() > j.GetID():
		return 1
	default:
		return 0
	}
}
