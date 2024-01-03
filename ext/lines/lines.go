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
	"io/fs"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/osutil"
)

// Register registers the lines and lines_read table-valued functions.
// The lines function reads from a database blob or text.
// The lines_read function reads from a file or an [io.Reader].
// If a filename is specified, [os.Open] is used to open the file.
func Register(db *sqlite3.Conn) {
	RegisterFS(db, osutil.FS{})
}

// RegisterFS registers the lines and lines_read table-valued functions.
// The lines function reads from a database blob or text.
// The lines_read function reads from a file or an [io.Reader].
// If a filename is specified, fsys is used to open the file.
func RegisterFS(db *sqlite3.Conn, fsys fs.FS) {
	sqlite3.CreateModule[lines](db, "lines", nil,
		func(db *sqlite3.Conn, _, _, _ string, _ ...string) (lines, error) {
			err := db.DeclareVtab(`CREATE TABLE x(line TEXT, data HIDDEN)`)
			db.VtabConfig(sqlite3.VTAB_INNOCUOUS)
			return lines{}, err
		})
	sqlite3.CreateModule[lines](db, "lines_read", nil,
		func(db *sqlite3.Conn, _, _, _ string, _ ...string) (lines, error) {
			err := db.DeclareVtab(`CREATE TABLE x(line TEXT, data HIDDEN)`)
			db.VtabConfig(sqlite3.VTAB_DIRECTONLY)
			return lines{fsys}, err
		})
}

type lines struct {
	fsys fs.FS
}

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
	if l.fsys != nil {
		return &reader{fsys: l.fsys}, nil
	} else {
		return &buffer{}, nil
	}
}

type cursor struct {
	line  []byte
	rowID int64
	eof   bool
}

func (c *cursor) EOF() bool {
	return c.eof
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx *sqlite3.Context, n int) error {
	if n == 0 {
		ctx.ResultRawText(c.line)
	}
	return nil
}

type reader struct {
	fsys   fs.FS
	reader *bufio.Reader
	closer io.Closer
	cursor
}

func (c *reader) Close() (err error) {
	if c.closer != nil {
		err = c.closer.Close()
		c.closer = nil
	}
	return err
}

func (c *reader) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	if err := c.Close(); err != nil {
		return err
	}

	var r io.Reader
	typ := arg[0].Type()
	switch typ {
	case sqlite3.NULL:
		if p, ok := arg[0].Pointer().(io.Reader); ok {
			r = p
		}
	case sqlite3.TEXT:
		f, err := c.fsys.Open(arg[0].Text())
		if err != nil {
			return err
		}
		r = f
	}
	if r == nil {
		return fmt.Errorf("lines: unsupported argument:%.0w %v", sqlite3.MISMATCH, typ)
	}

	c.reader = bufio.NewReader(r)
	c.closer, _ = r.(io.Closer)
	c.rowID = 0
	return c.Next()
}

func (c *reader) Next() (err error) {
	c.line = c.line[:0]
	for more := true; more; {
		var line []byte
		line, more, err = c.reader.ReadLine()
		c.line = append(c.line, line...)
	}
	if err == io.EOF {
		c.eof = true
		err = nil
	}
	c.rowID++
	return err
}

type buffer struct {
	data []byte
	cursor
}

func (c *buffer) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	typ := arg[0].Type()
	switch typ {
	case sqlite3.TEXT:
		c.data = arg[0].RawText()
	case sqlite3.BLOB:
		c.data = arg[0].RawBlob()
	default:
		return fmt.Errorf("lines: unsupported argument:%.0w %v", sqlite3.MISMATCH, typ)
	}

	c.rowID = 0
	return c.Next()
}

func (c *buffer) Next() error {
	i := bytes.IndexByte(c.data, '\n')
	j := i + 1
	switch {
	case i < 0:
		i = len(c.data)
		j = i
	case i > 0 && c.data[i-1] == '\r':
		i--
	}
	c.eof = len(c.data) == 0
	c.line = c.data[:i]
	c.data = c.data[j:]
	c.rowID++
	return nil
}
