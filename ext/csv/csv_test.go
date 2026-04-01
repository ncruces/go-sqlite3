package csv_test

import (
	"os"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/csv"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(csv.Register)
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const data = `
# Comment
"Rob"	"Pike"	rob
"Ken"	Thompson	ken
Robert	"Griesemer"	"gri"`
	err = db.Exec(`
		CREATE VIRTUAL TABLE temp.users USING csv(
			data    = ` + sqlite3.Quote(data) + `,
			schema  = 'CREATE TABLE x(first_name, last_name, username)',
			comma   = '\t',
			comment = '#'
		)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT * FROM temp.users WHERE rowid = 1 ORDER BY username`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal("no rows")
	}
	if got := stmt.ColumnText(0); got != "Rob" {
		t.Errorf("got %q want Rob", got)
	}
	if stmt.Step() {
		t.Fatal("more rows")
	}

	err = db.Exec(`ALTER TABLE temp.users RENAME TO csv`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`PRAGMA integrity_check`)
	if err != nil {
		t.Error(err)
	}

	err = db.Exec(`PRAGMA quick_check`)
	if err != nil {
		t.Error(err)
	}

	err = db.Exec(`DROP TABLE temp.csv`)
	if err != nil {
		t.Error(err)
	}
}

func TestAffinity(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const data = "01\n0.10\ne"
	err = db.Exec(`
		CREATE VIRTUAL TABLE temp.nums USING csv(
			data   = ` + sqlite3.Quote(data) + `,
			schema = 'CREATE TABLE x(a numeric)'
		)`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT * FROM temp.nums`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else if got := stmt.ColumnText(0); got != "1" {
		t.Errorf("got %q want 1", got)
	}
	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else if got := stmt.ColumnText(0); got != "0.1" {
		t.Errorf("got %q want 0.1", got)
	}

	if !stmt.Step() {
		t.Fatal(stmt.Err())
	} else if got := stmt.ColumnText(0); got != "e" {
		t.Errorf("got %q want e", got)
	}
}

func TestRegister_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE VIRTUAL TABLE temp.users USING csv()`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE temp.users USING csv(data='abc', data='abc')`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE temp.users USING csv(data='abc', xpto='abc')`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE temp.users USING csv(data='abc', comma='"')`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}

	err = db.Exec(`CREATE VIRTUAL TABLE temp.users USING csv(data='abc', header=tru)`)
	if err == nil {
		t.Fatal("want error")
	} else {
		t.Log(err)
	}
}
