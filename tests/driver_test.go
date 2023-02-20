package tests

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestDriver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS users (id INT, name VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := conn.ExecContext(ctx,
		`INSERT INTO users(id, name) VALUES(0, 'go'), (1, 'zig'), (2, 'whatever')`)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := res.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}
	if changes != 3 {
		t.Errorf("got %d want 3", changes)
	}

	stmt, err := conn.PrepareContext(context.Background(),
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
