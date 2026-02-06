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
		`CREATE TABLE t(a INT, b TEXT)`,
		`CREATE TABLE t1(
			id INTEGER PRIMARY KEY ASC,
			name TEXT DEFAULT (upper('x')),
			c TEXT CHECK((a+(b))) -- col comment
		) -- table comment`,
		`CREATE TEMP TABLE IF NOT EXISTS [w"eird]]t] ("q""q" INT)`,
		`ALTER TABLE t RENAME COLUMN a TO b;`,
		`ALTER TABLE main.t ADD COLUMN z INTEGER DEFAULT (1+(2*(3)))`,
		`/* cstyle */ CREATE TABLE x(y INT); -- tail`,
		`CREATE TABLE ct (d INT DEFAULT ( (1+2) ), e TEXT DEFAULT '))')`,

		`CREATE TABLE foo (col1 INTEGER PRIMARY KEY AUTOINCREMENT, col2 TEXT, col3 TEXT);`,
		`CREATE TABLE tcpkai (col INTEGER, PRIMARY KEY (col AUTOINCREMENT));`,
		`CREATE TABLE t1(x INTEGER PRIMARY KEY, y);`,
		`create table employee(first varchar(15),last varchar(20),age number(3),address varchar(30),city varchar(20),state varchar(20));`,
		`CREATE TEMP TABLE IF NOT EXISTS main.foo /* This is the main table */ (col1 INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, col2 TEXT DEFAULT CURRENT_TIMESTAMP, col3 FLOAT(8.12), col4 BLOB COLLATE BINARY /* Use this column for storing pictures */, CONSTRAINT tbl1 UNIQUE (col1 COLLATE c1 ASC, col2 COLLATE c2 DESC)) WITHOUT ROWID; -- this is a line comment`,
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
		`CREATE TABLE Sales(Price INT, Qty INT, Total INT GENERATED ALWAYS AS (Price*Qty) VIRTUAL, Item TEXT);`,
		`CREATE TABLE Constraints(
			PK  INTEGER CONSTRAINT 'PrimaryKey' PRIMARY KEY  CONSTRAINT 'NotNull' NOT NULL  CONSTRAINT 'Unique' UNIQUE
						CONSTRAINT 'Check'      CHECK (PK>0) CONSTRAINT 'Default' DEFAULT 2 CONSTRAINT 'Collate' COLLATE NOCASE,
			FK  INTEGER CONSTRAINT 'ForeignKey' REFERENCES ForeignTable (Id),
			GEN INTEGER CONSTRAINT 'Generated' AS (abs(PK)));`,
		`CREATE TABLE ColumnChecks(Num INT CONSTRAINT 'GT' CHECK (Num>0) CONSTRAINT 'LT' CHECK (Num<10) CONSTRAINT 'NE' CHECK(Num<>5));`,
		// GENERATED ALWAYS AS ... STORED
		`CREATE TABLE Inventory(Price REAL, Qty INT, Total REAL GENERATED ALWAYS AS (Price*Qty) STORED);`,
		// Table-level CHECK constraint
		`CREATE TABLE RangeCheck(lo INT, hi INT, CHECK (lo < hi));`,
		// Table-level CHECK constraint with name
		`CREATE TABLE NamedCheck(val INT, CONSTRAINT 'ValidRange' CHECK (val >= 0 AND val <= 100));`,
		// Composite PRIMARY KEY (table-level)
		`CREATE TABLE CompositePK(a INT, b TEXT, c REAL, PRIMARY KEY (a, b));`,
		// Composite PRIMARY KEY with ordering and collation
		`CREATE TABLE CompositePKOrder(x INT, y TEXT, PRIMARY KEY (x DESC, y COLLATE NOCASE ASC));`,
		// Table-level FOREIGN KEY with ON DELETE CASCADE and ON UPDATE SET NULL
		`CREATE TABLE Orders(id INTEGER PRIMARY KEY, customer_id INT, FOREIGN KEY (customer_id) REFERENCES Customers(id) ON DELETE CASCADE ON UPDATE SET NULL);`,
		// Table-level FOREIGN KEY with ON DELETE SET DEFAULT and ON UPDATE RESTRICT
		`CREATE TABLE LineItems(id INTEGER PRIMARY KEY, order_id INT, FOREIGN KEY (order_id) REFERENCES Orders(id) ON DELETE SET DEFAULT ON UPDATE RESTRICT);`,
		// Table-level FOREIGN KEY with ON DELETE NO ACTION and DEFERRABLE INITIALLY DEFERRED
		`CREATE TABLE Payments(id INTEGER PRIMARY KEY, order_id INT, FOREIGN KEY (order_id) REFERENCES Orders(id) ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED);`,
		// Table-level FOREIGN KEY with NOT DEFERRABLE INITIALLY IMMEDIATE
		`CREATE TABLE Shipments(id INTEGER PRIMARY KEY, order_id INT, FOREIGN KEY (order_id) REFERENCES Orders(id) ON DELETE RESTRICT NOT DEFERRABLE INITIALLY IMMEDIATE);`,
		// Column-level REFERENCES with ON DELETE CASCADE ON UPDATE SET NULL
		`CREATE TABLE Detail(id INTEGER PRIMARY KEY, parent_id INT REFERENCES Parent(id) ON DELETE CASCADE ON UPDATE SET NULL);`,
		// Multiple table constraints (PK + UNIQUE + CHECK + FK)
		`CREATE TABLE Multi(a INT, b INT, c INT, d INT,
			PRIMARY KEY (a, b),
			UNIQUE (c),
			CHECK (d > 0),
			FOREIGN KEY (d) REFERENCES Other(id) ON DELETE CASCADE);`,
		// ON CONFLICT clauses on column constraints
		`CREATE TABLE ConflictTest(
			a INT PRIMARY KEY ON CONFLICT ROLLBACK,
			b INT NOT NULL ON CONFLICT ABORT,
			c INT UNIQUE ON CONFLICT REPLACE);`,
		// ON CONFLICT clause on table-level PRIMARY KEY
		`CREATE TABLE ConflictPK(a INT, b INT, PRIMARY KEY (a, b) ON CONFLICT IGNORE);`,
		// ON CONFLICT clause on table-level UNIQUE
		`CREATE TABLE ConflictUniq(a INT, CONSTRAINT 'uniq1' UNIQUE (a) ON CONFLICT FAIL);`,
		// STRICT only (without WITHOUT ROWID)
		`CREATE TABLE StrictOnly(id INTEGER PRIMARY KEY, val TEXT) STRICT;`,
		// Backtick-quoted identifiers
		"CREATE TABLE `my table`(`col 1` INT, `col 2` TEXT);",
		// DEFAULT with negative number
		`CREATE TABLE Defaults(a INT DEFAULT -42, b REAL DEFAULT -3.14, c INT DEFAULT +0);`,
		// DEFAULT with various keywords
		`CREATE TABLE DefaultKeywords(a TEXT DEFAULT CURRENT_DATE, b TEXT DEFAULT CURRENT_TIME, c TEXT DEFAULT CURRENT_TIMESTAMP);`,
		// Single column table
		`CREATE TABLE Single(only_col INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL);`,
		// Column with no type
		`CREATE TABLE NoTypes(a, b, c);`,
		// PRIMARY KEY DESC with AUTOINCREMENT
		`CREATE TABLE PKDesc(id INTEGER PRIMARY KEY DESC);`,
		// Column-level foreign key with multiple columns in referenced table
		`CREATE TABLE FKMultiCol(id INTEGER PRIMARY KEY, a INT, b INT,
			FOREIGN KEY (a, b) REFERENCES Other(x, y) ON DELETE SET NULL ON UPDATE CASCADE);`,
		// DEFERRABLE without INITIALLY clause
		`CREATE TABLE DeferSimple(id INTEGER PRIMARY KEY, ref INT,
			FOREIGN KEY (ref) REFERENCES Parent(id) DEFERRABLE);`,
		// NOT DEFERRABLE INITIALLY DEFERRED
		`CREATE TABLE DeferNotDef(id INTEGER PRIMARY KEY, ref INT,
			FOREIGN KEY (ref) REFERENCES Parent(id) NOT DEFERRABLE INITIALLY DEFERRED);`,
		// Named table-level constraints
		`CREATE TABLE NamedConstraints(a INT, b INT, c INT,
			CONSTRAINT pk_named PRIMARY KEY (a),
			CONSTRAINT uq_named UNIQUE (b),
			CONSTRAINT ck_named CHECK (c != 0));`,
		// All constraint types combined on a single column
		`CREATE TABLE FullCol(x INTEGER PRIMARY KEY ON CONFLICT ABORT NOT NULL ON CONFLICT FAIL UNIQUE ON CONFLICT IGNORE
			CHECK (x > 0) DEFAULT 1 COLLATE BINARY REFERENCES Other(id));`,
		// Generated column shorthand (AS without GENERATED ALWAYS)
		`CREATE TABLE GenShort(a INT, b INT, c INT AS (a + b) STORED, d TEXT AS (a || b) VIRTUAL);`,
		// Mixed quoting styles
		"CREATE TABLE \"Mixed\"([col1] INT, `col2` TEXT, col3 BLOB);",
		// Table with many columns and types
		`CREATE TABLE AllTypes(a INTEGER, b REAL, c TEXT, d BLOB, e NUMERIC, f BOOLEAN, g DATE, h DATETIME, i DECIMAL(10,2), j VARCHAR(255));`,
		// Empty string and NULL defaults
		`CREATE TABLE EmptyDefaults(a TEXT DEFAULT '', b TEXT DEFAULT NULL, c INT DEFAULT 0);`,
		// Case insensitivity
		`create temporary table if not exists Foo(Bar integer primary key autoincrement, Baz text not null unique);`,
		// ALTER TABLE with schema-qualified name and ADD COLUMN with all constraints
		`ALTER TABLE main.foo ADD COLUMN new_col INTEGER NOT NULL DEFAULT 0 REFERENCES Other(id)`,

		// https://www.sqlite.org/lang_altertable.html
		`ALTER TABLE foo RENAME TO bar`,
		`ALTER TABLE temp.foo RENAME TO bar`,
		`ALTER TABLE foo RENAME COLUMN col1 TO col2`,
		`ALTER TABLE foo RENAME col1 TO col2`,
		`ALTER TABLE foo DROP COLUMN col1`,
		`ALTER TABLE foo ADD COLUMN col1 TEXT DEFAULT 'Hello' COLLATE NOCASE`,
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
