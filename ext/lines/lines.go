// Package lines provides a virtual table to read large files line-by-line.
package lines

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/ncruces/go-sqlite3"
)

// Register registers the lines and lines_read virtual tables.
// The lines virtual table reads from a database blob or text.
// The lines_read virtual table reads from a file or an [io.ReaderAt].
func Register(db *sqlite3.Conn) {
	sqlite3.CreateModule[lines](db, "lines", nil,
		func(db *sqlite3.Conn, arg ...string) (lines, error) {
			err := db.DeclareVtab(`CREATE TABLE x(line TEXT, data HIDDEN)`)
			db.VtabConfig(sqlite3.VTAB_INNOCUOUS)
			return false, err
		})
	sqlite3.CreateModule[lines](db, "lines_read", nil,
		func(db *sqlite3.Conn, arg ...string) (lines, error) {
			err := db.DeclareVtab(`CREATE TABLE x(line TEXT, data HIDDEN)`)
			db.VtabConfig(sqlite3.VTAB_DIRECTONLY)
			return true, err
		})
}

type lines bool

func (l lines) BestIndex(idx *sqlite3.IndexInfo) error {
	for i, cst := range idx.Constraint {
		if cst.Column == 1 && cst.Op == sqlite3.INDEX_CONSTRAINT_EQ && cst.Usable {
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 1,
			}
			idx.EstimatedCost = 1e6
			idx.EstimatedRows = 100
			return nil
		}
	}
	return sqlite3.CONSTRAINT
}

func (l lines) Open() (sqlite3.VTabCursor, error) {
	return &cursor{reader: bool(l)}, nil
}

type cursor struct {
	reader  bool
	scanner *bufio.Scanner
	closer  io.Closer
	rowID   int64
	eof     bool
}

func (c *cursor) Close() (err error) {
	if c.closer != nil {
		err = c.closer.Close()
		c.closer = nil
	}
	return err
}

func (c *cursor) EOF() bool {
	return c.eof
}

func (c *cursor) Next() error {
	c.rowID++
	c.eof = !c.scanner.Scan()
	return c.scanner.Err()
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx *sqlite3.Context, n int) error {
	if n == 0 {
		ctx.ResultRawText(c.scanner.Bytes())
	}
	return nil
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	if err := c.Close(); err != nil {
		return err
	}

	var r io.Reader
	data := arg[0]
	if c.reader {
		if data.Type() == sqlite3.NULL {
			if p, ok := data.Pointer().(io.ReaderAt); ok {
				r = io.NewSectionReader(p, 0, math.MaxInt64)
			}
		} else {
			f, err := os.Open(data.Text())
			if err != nil {
				return err
			}
			c.closer = f
			r = f
		}
	} else if data.Type() != sqlite3.NULL {
		r = bytes.NewReader(data.RawBlob())
	}

	if r == nil {
		return fmt.Errorf("lines: unsupported argument:%.0w %v", sqlite3.MISMATCH, data.Type())
	}
	c.scanner = bufio.NewScanner(r)
	c.rowID = 0
	return c.Next()
}
