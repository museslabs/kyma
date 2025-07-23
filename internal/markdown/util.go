package markdown

import "slices"

// PrioritizedValue holds a value along with its associated priority.
type PrioritizedValue[T any] struct {
	// Value is the item being prioritized.
	Value T
	// Priority is a priority of the value.
	Priority int
}

// PrioritizedSlice is a slice of [PrioritizedValue] items.
type PrioritizedSlice[T any] []PrioritizedValue[T]

// Sort arranges the slice in ascending order of priority.
func (s PrioritizedSlice[T]) Sort() {
	slices.SortFunc(s, func(a, b PrioritizedValue[T]) int {
		return a.Priority - b.Priority
	})
}

// Prioritized creates a new [PrioritizedValue] with the given value and priority.
func Prioritized[T any](v T, priority int) PrioritizedValue[T] {
	return PrioritizedValue[T]{v, priority}
}
