//go:build !unix && !windows

package sqlite3_wrap

type mmapState struct{}

func (s *mmapState) unmapAll() {}
