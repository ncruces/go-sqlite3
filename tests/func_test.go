package tests

import (
	"errors"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestCreateFunction(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.CreateFunction("test", 1, sqlite3.INNOCUOUS, func(ctx sqlite3.Context, arg ...sqlite3.Value) {
		switch arg := arg[0]; arg.Int() {
		case 0:
			ctx.ResultInt(arg.Int())
		case 1:
			ctx.ResultInt64(arg.Int64())
		case 2:
			ctx.ResultBool(arg.Bool())
		case 3:
			ctx.ResultFloat(arg.Float())
		case 4:
			ctx.ResultText(arg.Text())
		case 5:
			ctx.ResultBlob(arg.Blob(nil))
		case 6:
			ctx.ResultZeroBlob(arg.Int64())
		case 7:
			ctx.ResultTime(arg.Time(sqlite3.TimeFormatUnix), sqlite3.TimeFormatDefault)
		case 8:
			ctx.ResultNull()
		case 9:
			ctx.ResultError(sqlite3.FULL)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT test(value) FROM generate_series(0, 9)`)
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.INTEGER {
			t.Errorf("got %v, want INTEGER", got)
		}
		if got := stmt.ColumnInt(0); got != 0 {
			t.Errorf("got %v, want 1", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.INTEGER {
			t.Errorf("got %v, want INTEGER", got)
		}
		if got := stmt.ColumnInt64(0); got != 1 {
			t.Errorf("got %v, want 2", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.INTEGER {
			t.Errorf("got %v, want INTEGER", got)
		}
		if got := stmt.ColumnBool(0); got != true {
			t.Errorf("got %v, want true", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.FLOAT {
			t.Errorf("got %v, want FLOAT", got)
		}
		if got := stmt.ColumnInt64(0); got != 3 {
			t.Errorf("got %v, want 3", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.TEXT {
			t.Errorf("got %v, want TEXT", got)
		}
		if got := stmt.ColumnText(0); got != "4" {
			t.Errorf("got %s, want 4", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.BLOB {
			t.Errorf("got %v, want BLOB", got)
		}
		if got := stmt.ColumnRawBlob(0); string(got) != "5" {
			t.Errorf("got %s, want 5", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.BLOB {
			t.Errorf("got %v, want BLOB", got)
		}
		if got := stmt.ColumnRawBlob(0); len(got) != 6 {
			t.Errorf("got %v, want 6", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.TEXT {
			t.Errorf("got %v, want TEXT", got)
		}
		if got := stmt.ColumnTime(0, sqlite3.TimeFormatAuto); got.Unix() != 7 {
			t.Errorf("got %v, want 7", got)
		}
	}

	if stmt.Step() {
		if got := stmt.ColumnType(0); got != sqlite3.NULL {
			t.Errorf("got %v, want NULL", got)
		}
	}

	if stmt.Step() {
		t.Error("want error")
	}
	if err := stmt.Err(); !errors.Is(err, sqlite3.FULL) {
		t.Errorf("got %v, want sqlite3.FULL", err)
	}
}
