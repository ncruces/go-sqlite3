//go:build !(linux || darwin) || !(amd64 || arm64) || sqlite3_flock || sqlite3_nosys

package util

type mmapState struct{}

func (s mmapState) closeNotify() {}
