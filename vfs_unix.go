//go:build unix

package sqlite3

import "os"

func deleteOnClose(f *os.File) {
	_ = os.Remove(f.Name())
}
