package csv

type set[T comparable] map[T]struct{}

func (s set[T]) add(t T) {
	s[t] = struct{}{}
}

func (s set[T]) has(t T) bool {
	_, ok := s[t]
	return ok
}
