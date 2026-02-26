package closure_test

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/closure"
)

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(closure.Register)
	os.Exit(m.Run())
}

func Example() {
	db, err := sqlite3.Open(":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	closure.Register(db)

	err = db.Exec(`
		CREATE TABLE employees (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			name TEXT
		);
		CREATE INDEX employees_parent_idx ON employees(parent_id);
		INSERT INTO employees (id, parent_id, name) VALUES
			(11, NULL, 'Diane'),
			(12, 11, 'Bob'),
			(21, 11, 'Emma'),
			(22, 21, 'Grace'),
			(23, 21, 'Henry'),
			(24, 21, 'Irene'),
			(25, 21, 'Frank'),
			(31, 11, 'Cindy'),
			(32, 31, 'Dave'),
			(33, 31, 'Alice');
		CREATE VIRTUAL TABLE hierarchy USING transitive_closure(
			tablename = "employees",
			idcolumn = "id",
			parentcolumn = "parent_id"
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT employees.id, name FROM employees, hierarchy
		WHERE employees.id = hierarchy.id AND hierarchy.root = 31
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for stmt.Step() {
		fmt.Println(stmt.ColumnInt(0), stmt.ColumnText(1))
	}
	if err := stmt.Err(); err != nil {
		log.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		log.Fatal(err)
	}
	// Output:
	// 31 Cindy
	// 32 Dave
	// 33 Alice
}

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE TABLE employees (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			name TEXT
		);
		CREATE INDEX employees_parent_idx ON employees(parent_id);
		INSERT INTO employees (id, parent_id, name) VALUES
			(11, NULL, 'Diane'),
			(12, 11, 'Bob'),
			(21, 11, 'Emma'),
			(22, 21, 'Grace'),
			(23, 21, 'Henry'),
			(24, 21, 'Irene'),
			(25, 21, 'Frank'),
			(31, 11, 'Cindy'),
			(32, 31, 'Dave'),
			(33, 31, 'Alice');
		CREATE VIRTUAL TABLE temp.closure USING transitive_closure;
	`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`
		SELECT employees.id, name FROM employees, closure
		 WHERE employees.id = closure.id
		   AND closure.root = 31
		   AND closure.depth < 1
		   AND closure.tablename='employees'
		   AND closure.idcolumn='id'
		   AND closure.parentcolumn='parent_id'
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Error("want row")
	}
	if stmt.Step() {
		t.Error("don't want row")
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE VIRTUAL TABLE hierarchy USING transitive_closure(table='employees')`)
	if err == nil {
		t.Error("want error")
	}

	err = db.Exec(`CREATE VIRTUAL TABLE hierarchy USING transitive_closure(tablename='employees', tablename="employees")`)
	if err == nil {
		t.Error("want error")
	}

	err = db.Exec("CREATE VIRTUAL TABLE hierarchy USING transitive_closure(tablename=`employees`)")
	if err != nil {
		t.Error(err)
	}

	err = db.Exec(`SELECT * FROM hierarchy`)
	if err == nil {
		t.Error("want error")
	}
}
