package tests

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestDriver(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, nil, func(c *sqlite3.Conn) error {
		return c.Exec(`PRAGMA optimize`)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	res, err := conn.ExecContext(t.Context(),
		`CREATE TABLE users (id INTEGER PRIMARY KEY NOT NULL, name VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := res.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}
	if changes != 0 {
		t.Errorf("got %d want 0", changes)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if id != 0 {
		t.Errorf("got %d want 0", changes)
	}

	res, err = conn.ExecContext(t.Context(),
		`INSERT INTO users (id, name) VALUES (0, 'go'), (1, 'zig'), (2, 'whatever')`)
	if err != nil {
		t.Fatal(err)
	}
	changes, err = res.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}
	if changes != 3 {
		t.Errorf("got %d want 3", changes)
	}

	stmt, err := conn.PrepareContext(t.Context(),
		`SELECT id, name FROM users`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	typs, err := rows.ColumnTypes()
	if err != nil {
		t.Fatal(err)
	}
	if got := typs[0].DatabaseTypeName(); got != "INTEGER" {
		t.Errorf("got %s, want INTEGER", got)
	}
	if got := typs[1].DatabaseTypeName(); got != "VARCHAR" {
		t.Errorf("got %s, want VARCHAR", got)
	}
	if got, ok := typs[0].Nullable(); got || !ok {
		t.Errorf("got %v/%v, want false/true", got, ok)
	}
	if got, ok := typs[1].Nullable(); !got || ok {
		t.Errorf("got %v/%v, want true/false", got, ok)
	}

	row := 0
	ids := []int{0, 1, 2}
	names := []string{"go", "zig", "whatever"}
	for ; rows.Next(); row++ {
		var id int
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			t.Fatal(err)
		}

		if id != ids[row] {
			t.Errorf("got %d, want %d", id, ids[row])
		}
		if name != names[row] {
			t.Errorf("got %q, want %q", name, names[row])
		}
	}
	if row != 3 {
		t.Errorf("got %d, want %d", row, len(ids))
	}

	err = rows.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = conn.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}
