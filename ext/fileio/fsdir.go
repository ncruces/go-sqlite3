package fileio

import (
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

const (
	_COL_NAME = iota
	_COL_MODE
	_COL_MTIME
	_COL_DATA
	_COL_LEVEL
	_COL_ROOT
	_COL_BASE
)

type fsdir struct{ fsys fs.FS }

func (d fsdir) BestIndex(idx *sqlite3.IndexInfo) error {
	var levelOp sqlite3.IndexConstraintOp
	var root bool
	level := -1
	base := -1

	for i, cst := range idx.Constraint {
		switch cst.Column {
		case _COL_ROOT:
			if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
				return sqlite3.CONSTRAINT
			}
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 1,
			}
			root = true
		case _COL_BASE:
			if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
				return sqlite3.CONSTRAINT
			}
			base = i
		case _COL_LEVEL:
			if !cst.Usable {
				break
			}
			switch cst.Op {
			case sqlite3.INDEX_CONSTRAINT_EQ, sqlite3.INDEX_CONSTRAINT_LE, sqlite3.INDEX_CONSTRAINT_LT:
				levelOp = cst.Op
				level = i
			}
		}
	}
	if !root {
		return sqlite3.CONSTRAINT
	}

	args := 2
	idx.IdxNum = 0
	idx.EstimatedCost = 1e9
	if base >= 0 {
		idx.IdxNum |= 1
		idx.EstimatedCost /= 1e4
		idx.ConstraintUsage[base] = sqlite3.IndexConstraintUsage{
			Omit:      true,
			ArgvIndex: args,
		}
		args++
	}
	if level >= 0 {
		idx.EstimatedCost /= 1e4
		idx.IdxNum |= (args - 1) << 1
		if levelOp == sqlite3.INDEX_CONSTRAINT_LT {
			idx.IdxNum |= 1 << 3
		}
		idx.ConstraintUsage[level] = sqlite3.IndexConstraintUsage{
			Omit:      levelOp != sqlite3.INDEX_CONSTRAINT_EQ,
			ArgvIndex: args,
		}
	}
	return nil
}

func (d fsdir) Open() (sqlite3.VTabCursor, error) {
	return &cursor{fsdir: d}, nil
}

type cursor struct {
	fsdir
	base     string
	next     func() (entry, bool)
	stop     func()
	curr     entry
	eof      bool
	rowID    int64
	maxLevel int
}

type entry struct {
	fs.DirEntry
	err   error
	path  string
	level int
}

func (c *cursor) Close() error {
	if c.stop != nil {
		c.stop()
	}
	return nil
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	if err := c.Close(); err != nil {
		return err
	}

	root := arg[0].Text()
	if i := idxNum & 1; i > 0 {
		base := arg[i].Text()
		if c.fsys != nil {
			base = path.Clean(base) + "/"
		} else {
			base = filepath.Clean(base) + string(filepath.Separator)
		}
		root = base + root
		c.base = base
	}
	if i := idxNum >> 1; i > 0 {
		c.maxLevel = arg[i&3].Int() - i>>2
	}

	c.next, c.stop = iter.Pull(func(yield func(entry) bool) {
		var stack []string
		walkDir := func(p string, d fs.DirEntry, err error) error {
			level := len(stack)
			for level > 1 && !strings.HasPrefix(c.dir(p), stack[level-1]) {
				level--
			}
			stack = stack[:level]
			level++

			if !yield(entry{d, err, p, level}) {
				return fs.SkipAll
			}
			if d != nil && d.IsDir() {
				if 0 < c.maxLevel && c.maxLevel <= level {
					return fs.SkipDir
				}
				stack = append(stack, p)
			}
			return nil
		}
		if c.fsys != nil {
			fs.WalkDir(c.fsys, root, walkDir)
		} else {
			filepath.WalkDir(root, walkDir)
		}
	})
	c.eof = false
	c.rowID = 0
	return c.Next()
}

func (c *cursor) Next() error {
	curr, ok := c.next()
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

func (c *cursor) Column(ctx sqlite3.Context, n int) error {
	switch n {
	case _COL_NAME:
		name := strings.TrimPrefix(c.curr.path, c.base)
		ctx.ResultText(name)

	case _COL_MODE:
		i, err := c.curr.Info()
		if err != nil {
			return err
		}
		ctx.ResultInt64(int64(i.Mode()))

	case _COL_MTIME:
		i, err := c.curr.Info()
		if err != nil {
			return err
		}
		ctx.ResultTime(i.ModTime(), sqlite3.TimeFormatUnixFrac)

	case _COL_DATA:
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

	case _COL_LEVEL:
		ctx.ResultInt(c.curr.level)
	}
	return nil
}

func (c *cursor) dir(p string) string {
	var dir string
	if c.fsys != nil {
		dir, _ = path.Split(p)
	} else {
		dir, _ = filepath.Split(p)
	}
	if dir == "" {
		dir = "."
	}
	return dir
}
