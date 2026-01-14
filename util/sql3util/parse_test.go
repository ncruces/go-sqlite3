package sql3util_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func TestParse_references(t *testing.T) {
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

func TestParse_constraint(t *testing.T) {
	tab, err := sql3util.ParseTable(`CREATE TABLE child(x, y, PRIMARY KEY(x, y))`)
	if err != nil {
		t.Fatal(err)
	}

	if got := tab.Name; got != "child" {
		t.Errorf("got %s, want child", got)
	}

	if got := len(tab.Columns); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
	if got := tab.Columns[0].Name; got != "x" {
		t.Errorf("got %s, want x", got)
	}
	if got := tab.Columns[1].Name; got != "y" {
		t.Errorf("got %s, want y", got)
	}

	if got := len(tab.Constraints); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
	if got := tab.Constraints[0].Type; got != sql3util.TABLECONSTRAINT_PRIMARYKEY {
		t.Errorf("got %d, want primary key", got)
	}
	if got := len(tab.Constraints[0].IndexedColumns); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
	if got := tab.Constraints[0].IndexedColumns[0].Name; got != "x" {
		t.Errorf("got %s, want x", got)
	}
	if got := tab.Constraints[0].IndexedColumns[1].Name; got != "y" {
		t.Errorf("got %s, want y", got)
	}
}

func TestParse_foreign(t *testing.T) {
	tab, err := sql3util.ParseTable(`CREATE TABLE child(x, y, FOREIGN KEY (x, y) REFERENCES parent)`)
	if err != nil {
		t.Fatal(err)
	}

	if got := tab.Name; got != "child" {
		t.Errorf("got %s, want child", got)
	}

	if got := len(tab.Columns); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
	if got := tab.Columns[0].Name; got != "x" {
		t.Errorf("got %s, want x", got)
	}
	if got := tab.Columns[1].Name; got != "y" {
		t.Errorf("got %s, want y", got)
	}

	if got := len(tab.Constraints); got != 1 {
		t.Errorf("got %d, want 1", got)
	}
	if got := tab.Constraints[0].Type; got != sql3util.TABLECONSTRAINT_FOREIGNKEY {
		t.Errorf("got %d, want foreign key", got)
	}
	if got := len(tab.Constraints[0].ForeignKeyNames); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
	if got := tab.Constraints[0].ForeignKeyNames[0]; got != "x" {
		t.Errorf("got %s, want x", got)
	}
	if got := tab.Constraints[0].ForeignKeyNames[1]; got != "y" {
		t.Errorf("got %s, want y", got)
	}
}
