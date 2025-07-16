package markdown

import "slices"

type PrioritizedValue[T any] struct {
	Value    T
	Priority int
}

type PrioritizedSlice[T any] []PrioritizedValue[T]

func (s PrioritizedSlice[T]) Sort() {
	slices.SortFunc(s, func(a, b PrioritizedValue[T]) int {
		return a.Priority - b.Priority
	})
}

func Prioritized[T any](v T, priority int) PrioritizedValue[T] {
	return PrioritizedValue[T]{v, priority}
}
