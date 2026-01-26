package sql3util_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func TestParseTable_references(t *testing.T) {
	tab, err := sql3util.ParseTable("CREATE TABLE child(`x` INT REFERENCES parent)")
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
	if got := col.Type; got != "INT" {
		t.Errorf("got %s, want INT", got)
	}

	fk := col.ForeignKeyClause
	if got := fk.Table; got != "parent" {
		t.Errorf("got %s, want parent", got)
	}
}

func TestParseTable_constraint(t *testing.T) {
	tab, err := sql3util.ParseTable(`CREATE TABLE child('x', 'y', PRIMARY KEY('x', 'y'))`)
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

func TestParseTable_foreign(t *testing.T) {
	tab, err := sql3util.ParseTable(`CREATE TABLE child(x, y, FOREIGN KEY (x, y) REFERENCES "parent")`)
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

func TestParseTable_upstream(t *testing.T) {
	tests := []string{
		"CREATE TABLE t(a INT, b TEXT)",
		`CREATE TABLE t1(
			id INTEGER PRIMARY KEY ASC,
			name TEXT DEFAULT (upper('x')),
			c TEXT CHECK((a+(b))) -- col comment
		) -- table comment`,
		`CREATE TEMP TABLE IF NOT EXISTS [w"eird]]t] ("q""q" INT)`,
		"ALTER TABLE t RENAME COLUMN a TO b;",
		"ALTER TABLE main.t ADD COLUMN z INTEGER DEFAULT (1+(2*(3)))",
		"/* cstyle */ CREATE TABLE x(y INT); -- tail",
		"CREATE TABLE ct (d INT DEFAULT ( (1+2) ), e TEXT DEFAULT '))')",

		"CREATE TABLE foo (col1 INTEGER PRIMARY KEY AUTOINCREMENT, col2 TEXT, col3 TEXT);",
		"CREATE TABLE tcpkai (col INTEGER, PRIMARY KEY (col AUTOINCREMENT));",
		"CREATE TABLE t1(x INTEGER PRIMARY KEY, y);",
		"create table employee(first varchar(15),last varchar(20),age number(3),address varchar(30),city varchar(20),state varchar(20));",
		"CREATE TEMP TABLE IF NOT EXISTS main.foo /* This is the main table */ (col1 INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, col2 TEXT DEFAULT CURRENT_TIMESTAMP, col3 FLOAT(8.12), col4 BLOB COLLATE BINARY /* Use this column for storing pictures */, CONSTRAINT tbl1 UNIQUE (col1 COLLATE c1 ASC, col2 COLLATE c2 DESC)) WITHOUT ROWID; -- this is a line comment",
		`CREATE TABLE "BalancesTbl2" ("id" INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL UNIQUE,  "checkingBal" REAL DEFAULT 0,  "cashBal" REAL DEFAULT .0,  "defitCardBal" REAL DEFAULT 1.0,  "creditCardBal" REAL DEFAULT +1.5,  testValue TEXT DEFAULT 'Hello World',   testValue2 TEXT DEFAULT 'Hello''s World', testValue3 TEXT DEFAULT "Hello''s World", testValue4 TEXT DEFAULT "Hello"" World") WITHOUT ROWID, STRICT;`,
		`CREATE TABLE User
			-- A table comment
			(
			uid INTEGER,    -- A field comment
			flags INTEGER,  -- Another field comment
			test TEXT /* Another C style comment */
			);`,
		`CREATE TABLE User
			-- A table comment
			(
			uid INTEGER,    -- A field comment
	    	flags /*This is another column comment*/ INTEGER   -- Another field comment
			, test -- test 123
			INTEGER, UNIQUE (flags /* Hello World*/, test) -- This is another table comment
			);`,
		"CREATE TABLE Sales(Price INT, Qty INT, Total INT GENERATED ALWAYS AS (Price*Qty) VIRTUAL, Item TEXT);",
		`CREATE TABLE Constraints(
			PK  INTEGER CONSTRAINT 'PrimaryKey' PRIMARY KEY  CONSTRAINT 'NotNull' NOT NULL  CONSTRAINT 'Unique' UNIQUE
						CONSTRAINT 'Check'      CHECK (PK>0) CONSTRAINT 'Default' DEFAULT 2 CONSTRAINT 'Collate' COLLATE NOCASE,
			FK  INTEGER CONSTRAINT 'ForeignKey' REFERENCES ForeignTable (Id),
			GEN INTEGER CONSTRAINT 'Generated' AS (abs(PK)));`,

		// https://www.sqlite.org/lang_altertable.html
		"ALTER TABLE foo RENAME TO bar",
		"ALTER TABLE temp.foo RENAME TO bar",
		"ALTER TABLE foo RENAME COLUMN col1 TO col2",
		"ALTER TABLE foo RENAME col1 TO col2",
		"ALTER TABLE foo DROP COLUMN col1",
		"ALTER TABLE foo ADD COLUMN col1 TEXT DEFAULT 'Hello' COLLATE NOCASE",
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			_, gotErr := sql3util.ParseTable(tt)
			if gotErr != nil {
				t.Errorf("ParseTable() failed: %v", gotErr)
			}
		})
	}
}
