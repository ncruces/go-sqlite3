//go:build !unix || sqlite3_noshm || sqlite3_nosys

package util

type mmapState struct{}
