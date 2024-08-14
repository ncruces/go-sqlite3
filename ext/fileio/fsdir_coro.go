//go:build !(go1.23 || goexperiment.rangefunc) || vet

package fileio

import (
	"io/fs"
	"path/filepath"
)

type resume = func(struct{}) (entry, bool)

func next(c *cursor) (entry, bool) {
	return c.resume(struct{}{})
}

func pull(c *cursor, root string) (resume, func()) {
	return coroNew(func(_ struct{}, yield func(entry) struct{}) entry {
		walkDir := func(path string, d fs.DirEntry, err error) error {
			yield(entry{d, err, path})
			return nil
		}
		if c.fsys != nil {
			fs.WalkDir(c.fsys, root, walkDir)
		} else {
			filepath.WalkDir(root, walkDir)
		}
		return entry{}
	})
}
