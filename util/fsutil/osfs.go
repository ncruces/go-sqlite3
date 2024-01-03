// Package fsutil implements file system utility functions.
package fsutil

import (
	"io/fs"
	"os"
)

// OSFS implements [fs.FS], [fs.StatFS], and [fs.ReadFileFS]
// using package [os].
//
// This filesystem does not respect [fs.ValidPath] rules,
// and fails [testing/fstest.TestFS]!
//
// Still, it can be a useful tool to unify implementations
// that can access either the [os] filesystem or an [fs.FS].
// It's OK to use this to open files, but you should avoid
// opening directories, resolving paths, or walking the file system.
type OSFS struct{}

// Open implements [fs.FS].
func (OSFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

// ReadFileFS implements [fs.StatFS].
func (OSFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile implements [fs.ReadFileFS].
func (OSFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
