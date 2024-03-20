//go:build !linux || !(amd64 || arm64) || sqlite3_nosys

package vfs

import "os"

func osBatchAtomic(*os.File) bool {
	return false
}
