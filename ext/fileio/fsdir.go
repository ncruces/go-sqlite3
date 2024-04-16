package fileio

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

type fsdir struct{ fsys fs.FS }

func (d fsdir) BestIndex(idx *sqlite3.IndexInfo) error {
	var root, base bool
	for i, cst := range idx.Constraint {
		switch cst.Column {
		case 4: // root
			if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
				return sqlite3.CONSTRAINT
			}
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 1,
			}
			root = true
		case 5: // base
			if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
				return sqlite3.CONSTRAINT
			}
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 2,
			}
			base = true
		}
	}
	if !root {
		return sqlite3.CONSTRAINT
	}
	if base {
		idx.EstimatedCost = 10
	} else {
		idx.EstimatedCost = 100
	}
	return nil
}

func (d fsdir) Open() (sqlite3.VTabCursor, error) {
	return &cursor{fsdir: d}, nil
}

type cursor struct {
	fsdir
	base   string
	resume func(struct{}) (entry, bool)
	cancel func()
	curr   entry
	eof    bool
	rowID  int64
}

type entry struct {
	fs.DirEntry
	err  error
	path string
}

func (c *cursor) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	if err := c.Close(); err != nil {
		return err
	}

	root := arg[0].Text()
	if len(arg) > 1 {
		base := arg[1].Text()
		if c.fsys != nil {
			root = path.Join(base, root)
			base = path.Clean(base) + "/"
		} else {
			root = filepath.Join(base, root)
			base = filepath.Clean(base) + string(filepath.Separator)
		}
		c.base = base
	}

	c.resume, c.cancel = coroNew(func(_ struct{}, yield func(entry) struct{}) entry {
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
	c.eof = false
	c.rowID = 0
	return c.Next()
}

func (c *cursor) Next() error {
	curr, ok := c.resume(struct{}{})
	c.curr = curr
	c.eof = !ok
	c.rowID++
	return c.curr.err
}

func (c *cursor) EOF() bool {
	return c.eof
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx *sqlite3.Context, n int) error {
	switch n {
	case 0: // name
		name := strings.TrimPrefix(c.curr.path, c.base)
		ctx.ResultText(name)

	case 1: // mode
		i, err := c.curr.Info()
		if err != nil {
			return err
		}
		ctx.ResultInt64(int64(i.Mode()))

	case 2: // mtime
		i, err := c.curr.Info()
		if err != nil {
			return err
		}
		ctx.ResultTime(i.ModTime(), sqlite3.TimeFormatUnixFrac)

	case 3: // data
		switch typ := c.curr.Type(); {
		case typ.IsRegular():
			var data []byte
			var err error
			if c.fsys != nil {
				data, err = fs.ReadFile(c.fsys, c.curr.path)
			} else {
				data, err = os.ReadFile(c.curr.path)
			}
			if err != nil {
				return err
			}
			ctx.ResultBlob(data)

		case typ&fs.ModeSymlink != 0 && c.fsys == nil:
			t, err := os.Readlink(c.curr.path)
			if err != nil {
				return err
			}
			ctx.ResultText(t)
		}
	}
	return nil
}
