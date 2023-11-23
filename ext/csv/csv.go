// Package csv provides a CSV virtual table.
//
// The CSV virtual table reads RFC 4180 formatted comma-separated values,
// and returns that content as if it were rows and columns of an SQL table.
//
// https://sqlite.org/csv.html
package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/ncruces/go-sqlite3"
)

// Register registers the CSV virtual table.
// If a filename is specified, `os.Open` is used to read it from disk.
func Register(db *sqlite3.Conn) {
	RegisterOpen(db, func(name string) (io.ReaderAt, error) {
		return os.Open(name)
	})
}

// RegisterOpen registers the CSV virtual table.
// If a filename is specified, open is used to open the file.
func RegisterOpen(db *sqlite3.Conn, open func(name string) (io.ReaderAt, error)) {
	declare := func(db *sqlite3.Conn, arg ...string) (_ *table, err error) {
		var (
			filename string
			data     string
			schema   string
			header   bool
			columns  int  = -1
			comma    rune = ','

			done = map[string]struct{}{}
		)

		for _, arg := range arg[3:] {
			key, val := getParam(arg)
			if _, ok := done[key]; ok {
				return nil, fmt.Errorf("csv: more than one %q parameter", key)
			}
			switch key {
			case "filename":
				filename = unquoteParam(val)
			case "data":
				data = unquoteParam(val)
			case "schema":
				schema = unquoteParam(val)
			case "header":
				header, err = boolParam(key, val)
			case "columns":
				columns, err = uintParam(key, val)
			case "comma":
				comma, err = runeParam(key, val)
			default:
				return nil, fmt.Errorf("csv: unknown %q parameter", key)
			}
			if err != nil {
				return nil, err
			}
			done[key] = struct{}{}
		}

		if (filename == "") == (data == "") {
			return nil, fmt.Errorf(`csv: must specify either "filename" or "data" but not both`)
		}

		var r io.ReaderAt
		if filename != "" {
			r, err = open(filename)
		} else {
			r = strings.NewReader(data)
		}
		if err != nil {
			return nil, err
		}

		table := &table{
			r:      r,
			comma:  comma,
			header: header,
		}
		defer func() {
			if err != nil {
				table.Close()
			}
		}()

		if schema == "" && (header || columns < 0) {
			csv := table.newReader()
			row, err := csv.Read()
			if err != nil {
				return nil, err
			}
			schema = getSchema(header, columns, row)
		}

		err = db.DeclareVtab(schema)
		if err != nil {
			return nil, err
		}
		err = db.VtabConfig(sqlite3.VTAB_DIRECTONLY)
		if err != nil {
			return nil, err
		}
		return table, nil
	}

	sqlite3.CreateModule(db, "csv", declare, declare)
}

type table struct {
	r      io.ReaderAt
	comma  rune
	header bool
}

func (t *table) Close() error {
	if c, ok := t.r.(io.Closer); ok {
		err := c.Close()
		t.r = nil
		return err
	}
	return nil
}

func (t *table) BestIndex(idx *sqlite3.IndexInfo) error {
	idx.EstimatedCost = 1e6
	return nil
}

func (t *table) Open() (sqlite3.VTabCursor, error) {
	return &cursor{table: t}, nil
}

func (t *table) Rename(new string) error {
	return nil
}

func (t *table) Integrity(schema, table string, flags int) (err error) {
	if flags&1 == 0 {
		_, err = t.newReader().ReadAll()
	}
	return err
}

func (t *table) newReader() *csv.Reader {
	csv := csv.NewReader(io.NewSectionReader(t.r, 0, math.MaxInt64))
	csv.ReuseRecord = true
	csv.Comma = t.comma
	return csv
}

type cursor struct {
	table *table
	rowID int64
	row   []string
	csv   *csv.Reader
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	c.csv = c.table.newReader()
	if c.table.header {
		c.Next() // skip header
	}
	c.rowID = 0
	return c.Next()
}

func (c *cursor) Next() (err error) {
	c.rowID++
	c.row, err = c.csv.Read()
	if err != io.EOF {
		return err
	}
	return nil
}

func (c *cursor) EOF() bool {
	return c.row == nil
}

func (c *cursor) RowID() (int64, error) {
	return c.rowID, nil
}

func (c *cursor) Column(ctx *sqlite3.Context, col int) error {
	if col < len(c.row) {
		ctx.ResultText(c.row[col])
	}
	return nil
}
