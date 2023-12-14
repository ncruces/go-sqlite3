// Package lines provides a virtual table to read data line-by-line.
//
// It is particularly useful for line-oriented datasets,
// like [ndjson] or [JSON Lines],
// when paired with SQLite's JSON support.
//
// https://github.com/asg017/sqlite-lines
//
// [ndjson]: https://ndjson.org/
// [JSON Lines]: https://jsonlines.org/
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
		func(db *sqlite3.Conn, _, _, _ string, _ ...string) (lines, error) {
			err := db.DeclareVtab(`CREATE TABLE x(line TEXT, data HIDDEN)`)
			db.VtabConfig(sqlite3.VTAB_INNOCUOUS)
			return false, err
		})
	sqlite3.CreateModule[lines](db, "lines_read", nil,
		func(db *sqlite3.Conn, _, _, _ string, _ ...string) (lines, error) {
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
	scanner *bufio.Scanner
	closer  io.Closer
	rowID   int64
	eof     bool
	reader  bool
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
	typ := data.Type()
	if c.reader {
		switch typ {
		case sqlite3.NULL:
			if p, ok := data.Pointer().(io.ReaderAt); ok {
				r = io.NewSectionReader(p, 0, math.MaxInt64)
			}
		case sqlite3.TEXT:
			f, err := os.Open(data.Text())
			if err != nil {
				return err
			}
			c.closer = f
			r = f
		}
	} else {
		switch typ {
		case sqlite3.TEXT:
			r = bytes.NewReader(data.RawText())
		case sqlite3.BLOB:
			r = bytes.NewReader(data.RawBlob())
		}
	}

	if r == nil {
		return fmt.Errorf("lines: unsupported argument:%.0w %v", sqlite3.MISMATCH, typ)
	}
	c.scanner = bufio.NewScanner(r)
	c.rowID = 0
	return c.Next()
}
