package tests

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/tests/testcfg"
)

func Test_base64(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// base64
	stmt, _, err := db.Prepare(`SELECT base64('TWFueSBoYW5kcyBtYWtlIGxpZ2h0IHdvcmsu')`)
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal("expected one row")
	}
	if got := stmt.ColumnText(0); got != "Many hands make light work." {
		t.Errorf("got %q", got)
	}
}

func Test_decimal(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT decimal_add(decimal('0.1'), decimal('0.2')) = decimal('0.3')`)
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal("expected one row")
	}
	if !stmt.ColumnBool(0) {
		t.Error("want true")
	}
}

func Test_uint(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT 'z2' < 'z11' COLLATE UINT`)
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal("expected one row")
	}
	if !stmt.ColumnBool(0) {
		t.Error("want true")
	}
}
