package fileio

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

type fsdir struct{ fs.FS }

func (d fsdir) BestIndex(idx *sqlite3.IndexInfo) error {
	var path, dir bool
	for i, cst := range idx.Constraint {
		switch cst.Column {
		case 4: // path
			if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
				return sqlite3.CONSTRAINT
			}
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 1,
			}
			path = true
		case 5: // dir
			if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
				return sqlite3.CONSTRAINT
			}
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 2,
			}
			dir = true
		}
	}
	if path {
		idx.EstimatedCost = 100
	}
	if dir {
		idx.EstimatedCost = 10
	}
	return nil
}

func (d fsdir) Open() (sqlite3.VTabCursor, error) {
	return &cursor{fs: d.FS}, nil
}

type cursor struct {
	fs    fs.FS
	dir   string
	rowID int64
	eof   bool
	curr  entry
	next  chan entry
	done  chan struct{}
}

type entry struct {
	path  string
	entry fs.DirEntry
	err   error
}

func (c *cursor) Close() error {
	if c.done != nil {
		close(c.done)
		s := <-c.next
		c.done = nil
		c.next = nil
		return s.err
	}
	return nil
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	if err := c.Close(); err != nil {
		return err
	}
	if len(arg) == 0 {
		return fmt.Errorf("fsdir: wrong number of arguments")
	}

	path := arg[0].Text()
	if len(arg) > 1 {
		if dir := arg[1].RawText(); c.fs != nil {
			c.dir = string(dir) + "/"
		} else {
			c.dir = string(dir) + string(filepath.Separator)
		}
		path = c.dir + path
	}

	c.rowID = 0
	c.eof = false
	c.next = make(chan entry)
	c.done = make(chan struct{})
	go c.WalkDir(path)
	return c.Next()
}

func (c *cursor) Next() error {
	curr, ok := <-c.next
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
		name := strings.TrimPrefix(c.curr.path, c.dir)
		ctx.ResultText(name)

	case 1: // mode
		i, err := c.curr.entry.Info()
		if err != nil {
			return err
		}
		ctx.ResultInt64(int64(i.Mode()))

	case 2: // mtime
		i, err := c.curr.entry.Info()
		if err != nil {
			return err
		}
		ctx.ResultTime(i.ModTime(), sqlite3.TimeFormatUnixFrac)

	case 3: // data
		switch typ := c.curr.entry.Type(); {
		case typ.IsRegular():
			var data []byte
			var err error
			if name := c.curr.entry.Name(); c.fs != nil {
				data, err = fs.ReadFile(c.fs, name)
			} else {
				data, err = os.ReadFile(name)
			}
			if err != nil {
				return err
			}
			ctx.ResultBlob(data)

		case typ&fs.ModeSymlink != 0 && c.fs == nil:
			t, err := os.Readlink(c.curr.entry.Name())
			if err != nil {
				return err
			}
			ctx.ResultText(t)
		}
	}
	return nil
}

func (c *cursor) WalkDir(path string) {
	var err error

	defer func() {
		if p := recover(); p != nil {
			if perr, ok := p.(error); ok {
				err = fmt.Errorf("panic: %w", perr)
			} else {
				err = fmt.Errorf("panic: %v", p)
			}
		}
		if err != nil {
			c.next <- entry{err: err}
		}
		close(c.next)
	}()

	if c.fs != nil {
		err = fs.WalkDir(c.fs, path, c.WalkDirFunc)
	} else {
		err = filepath.WalkDir(path, c.WalkDirFunc)
	}
}

func (c *cursor) WalkDirFunc(path string, de fs.DirEntry, err error) error {
	select {
	case <-c.done:
		return fs.SkipAll
	case c.next <- entry{path, de, err}:
		return nil
	}
}
