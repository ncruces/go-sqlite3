//go:build js && wasm

package sqlite3

import "os"

func init() {
	const dir = "/tmp"
	ensureTempDir(os.Getenv("TMPDIR"))
	ensureTempDir(os.Getenv("TMP"))
	ensureTempDir(os.Getenv("TEMP"))
	ensureTempDir(os.TempDir())
	if !ensureTempDir(dir) {
		return
	}
	_ = os.Setenv("TMPDIR", dir)
	_ = os.Setenv("TMP", dir)
	_ = os.Setenv("TEMP", dir)
}

func ensureTempDir(dir string) bool {
	if dir == "" {
		return false
	}
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return false
	}
	return true
}
