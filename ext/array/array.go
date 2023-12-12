// Package array provides the array table-valued SQL function.
//
// https://sqlite.org/carray.html
package array

import (
	"fmt"
	"reflect"

	"github.com/ncruces/go-sqlite3"
)

// Register registers the array single-argument, table-valued SQL function.
// The argument must be an [sqlite3.Pointer] to a Go slice or array
// of ints, floats, bools, strings or blobs.
func Register(db *sqlite3.Conn) {
	sqlite3.CreateModule[array](db, "array", nil,
		func(db *sqlite3.Conn, _, _, _ string, _ ...string) (array, error) {
			err := db.DeclareVtab(`CREATE TABLE x(value, array HIDDEN)`)
			return array{}, err
		})
}

type array struct{}

func (array) BestIndex(idx *sqlite3.IndexInfo) error {
	for i, cst := range idx.Constraint {
		if cst.Column == 1 && cst.Op == sqlite3.INDEX_CONSTRAINT_EQ && cst.Usable {
			idx.ConstraintUsage[i] = sqlite3.IndexConstraintUsage{
				Omit:      true,
				ArgvIndex: 1,
			}
			idx.EstimatedCost = 1
			idx.EstimatedRows = 100
			return nil
		}
	}
	return sqlite3.CONSTRAINT
}

func (array) Open() (sqlite3.VTabCursor, error) {
	return &cursor{}, nil
}

type cursor struct {
	array reflect.Value
	rowID int
}

func (c *cursor) EOF() bool {
	return c.rowID >= c.array.Len()
}

func (c *cursor) Next() error {
	c.rowID++
	return nil
}

func (c *cursor) RowID() (int64, error) {
	return int64(c.rowID), nil
}

func (c *cursor) Column(ctx *sqlite3.Context, n int) error {
	if n != 0 {
		return nil
	}

	v := c.array.Index(c.rowID)
	k := v.Kind()

	if k == reflect.Interface {
		if v.IsNil() {
			ctx.ResultNull()
			return nil
		}
		v = v.Elem()
		k = v.Kind()
	}

	switch {
	case v.CanInt():
		ctx.ResultInt64(v.Int())

	case v.CanUint():
		i64 := int64(v.Uint())
		if i64 < 0 {
			return fmt.Errorf("array: integer element overflow:%.0w %d", sqlite3.MISMATCH, v.Uint())
		}
		ctx.ResultInt64(i64)

	case v.CanFloat():
		ctx.ResultFloat(v.Float())

	case k == reflect.Bool:
		ctx.ResultBool(v.Bool())

	case k == reflect.String:
		ctx.ResultText(v.String())

	case (k == reflect.Slice || k == reflect.Array && v.CanAddr()) &&
		v.Type().Elem().Kind() == reflect.Uint8:
		ctx.ResultBlob(v.Bytes())

	default:
		return fmt.Errorf("array: unsupported element:%.0w %v", sqlite3.MISMATCH, v.Type())
	}
	return nil
}

func (c *cursor) Filter(idxNum int, idxStr string, arg ...sqlite3.Value) error {
	array := reflect.ValueOf(arg[0].Pointer())
	array, err := indexable(array)
	if err != nil {
		return err
	}

	c.array = array
	c.rowID = 0
	return nil
}

func indexable(v reflect.Value) (reflect.Value, error) {
	if v.Kind() == reflect.Slice {
		return v, nil
	}
	if v.Kind() == reflect.Array {
		return v, nil
	}
	if v.Kind() == reflect.Pointer {
		if v := v.Elem(); v.Kind() == reflect.Array {
			return v, nil
		}
	}
	return v, fmt.Errorf("array: unsupported argument:%.0w %v", sqlite3.MISMATCH, v)
}
