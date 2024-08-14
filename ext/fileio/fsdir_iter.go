//go:build (go1.23 || goexperiment.rangefunc) && !vet

package fileio

import (
	"io/fs"
	"iter"
	"path/filepath"
)

type resume = func() (entry, bool)

func next(c *cursor) (entry, bool) {
	return c.resume()
}

func pull(c *cursor, root string) (resume, func()) {
	return iter.Pull(func(yield func(entry) bool) {
		walkDir := func(path string, d fs.DirEntry, err error) error {
			if yield(entry{d, err, path}) {
				return nil
			}
			return fs.SkipAll
		}
		if c.fsys != nil {
			fs.WalkDir(c.fsys, root, walkDir)
		} else {
			filepath.WalkDir(root, walkDir)
		}
	})
}
