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
	"errors"
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
func Register(db *sqlite3.Conn) error {
	return RegisterFS(db, osutil.FS{})
}

// RegisterFS registers the lines and lines_read table-valued functions.
// The lines function reads from a database blob or text.
// The lines_read function reads from a file or an [io.Reader].
// If a filename is specified, fsys is used to open the file.
func RegisterFS(db *sqlite3.Conn, fsys fs.FS) error {
	return errors.Join(
		sqlite3.CreateModule(db, "lines", nil,
			func(db *sqlite3.Conn, _, _, _ string, _ ...string) (lines, error) {
				err := db.DeclareVTab(`CREATE TABLE x(line TEXT, data HIDDEN, delim HIDDEN)`)
				if err == nil {
					err = db.VTabConfig(sqlite3.VTAB_INNOCUOUS)
				}
				return lines{}, err
			}),
		sqlite3.CreateModule(db, "lines_read", nil,
			func(db *sqlite3.Conn, _, _, _ string, _ ...string) (lines, error) {
				err := db.DeclareVTab(`CREATE TABLE x(line TEXT, data HIDDEN, delim HIDDEN)`)
				if err == nil {
					err = db.VTabConfig(sqlite3.VTAB_DIRECTONLY)
				}
				return lines{fsys}, err
			}))
}

type lines struct {
	fsys fs.FS
}

func (l lines) BestIndex(idx *sqlite3.IndexInfo) (err error) {
	err = sqlite3.CONSTRAINT
	for i, cst := range idx.Constraint {
		if !cst.Usable || cst.Op != sqlite3.INDEX_CONSTRAINT_EQ {
			continue
		}
		switch cst.Column {
		case 1:
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 1,
			}
			idx.EstimatedCost = 1e6
			idx.EstimatedRows = 100
			err = nil
		case 2:
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 2,
			}
		}
	}
	return err
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
	delim byte
}

func (c *cursor) EOF() bool {
	return c.eof
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx sqlite3.Context, n int) error {
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

	c.delim = '\n'
	if len(arg) > 1 {
		b := arg[1].RawText()
		if len(b) != 1 {
			return fmt.Errorf("lines: delimiter must be a single byte%.0w", sqlite3.MISMATCH)
		}
		c.delim = b[0]
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
		if c.delim == '\n' {
			line, more, err = c.reader.ReadLine()
		} else {
			line, err = c.reader.ReadSlice(c.delim)
			more = err == bufio.ErrBufferFull
		}
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

	c.delim = '\n'
	if len(arg) > 1 {
		b := arg[1].RawText()
		if len(b) != 1 {
			return fmt.Errorf("lines: delimiter must be a single byte%.0w", sqlite3.MISMATCH)
		}
		c.delim = b[0]
	}

	c.rowID = 0
	return c.Next()
}

func (c *buffer) Next() error {
	i := bytes.IndexByte(c.data, c.delim)
	j := i + 1
	switch {
	case i < 0:
		i = len(c.data)
		j = i
	case i > 0 && c.delim == '\n' && c.data[i-1] == '\r':
		i--
	}
	c.eof = len(c.data) == 0
	c.line = c.data[:i]
	c.data = c.data[j:]
	c.rowID++
	return nil
}
