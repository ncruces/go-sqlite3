package sql3util_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func TestParse(t *testing.T) {
	tab, err := sql3util.ParseTable(`CREATE TABLE child(x REFERENCES parent)`)
	if err != nil {
		t.Fatal(err)
	}

	if got := tab.Name; got != "child" {
		t.Errorf("got %s, want child", got)
	}
	if got := len(tab.Columns); got != 1 {
		t.Errorf("got %d, want 1", got)
	}

	col := tab.Columns[0]
	if got := col.Name; got != "x" {
		t.Errorf("got %s, want x", got)
	}

	fk := col.ForeignKeyClause
	if got := fk.Table; got != "parent" {
		t.Errorf("got %s, want parent", got)
	}
}
