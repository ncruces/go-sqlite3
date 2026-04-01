package array_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/array"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test_cursor_Column(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	ctx := testcfg.Context(t)
	db, err := driver.Open(dsn, array.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT rowid, value FROM array(?)`,
		sqlite3.Pointer(&[...]any{nil, true, 1, uint(2), math.Pi, "text", []byte{1, 2, 3}}))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	want := []string{"nil", "int64", "int64", "int64", "float64", "string", "[]uint8"}

	for rows.Next() {
		var id, val any
		err := rows.Scan(&id, &val)
		if err != nil {
			t.Fatal(err)
		}
		if want := want[0]; val == nil {
			if want != "nil" {
				t.Errorf("got nil, want %s", want)
			}
		} else if got := reflect.TypeOf(val).String(); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
		want = want[1:]
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func Test_array_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = array.Register(db)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`SELECT * FROM array()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`SELECT * FROM array(?)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}
