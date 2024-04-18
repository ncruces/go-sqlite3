package util

func Unwrap[T any](v T) T {
	if u, ok := any(v).(interface{ Unwrap() T }); ok {
		return u.Unwrap()
	}
	return v
}
