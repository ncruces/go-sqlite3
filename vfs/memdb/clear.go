//go:build !go1.21

package memdb

func clear[T any](b []T) {
	var zero T
	for i := range b {
		b[i] = zero
	}
}
