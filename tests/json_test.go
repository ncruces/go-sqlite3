package tests

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/julianday"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := driver.Open(":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS test (col)`)
	if err != nil {
		t.Fatal(err)
	}

	reference := time.Date(2013, 10, 7, 4, 23, 19, 120_000_000, time.FixedZone("", -4*3600))

	_, err = conn.ExecContext(ctx,
		`INSERT INTO test (col) VALUES (?), (?), (?), (?)`,
		nil, 1, math.Pi, reference,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = conn.ExecContext(ctx,
		`INSERT INTO test (col) VALUES (?), (?), (?), (?)`,
		sqlite3.JSON(math.Pi), sqlite3.JSON(false),
		julianday.Format(reference), sqlite3.JSON([]string{}))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := conn.QueryContext(ctx, "SELECT * FROM test")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"null", "1", "3.141592653589793",
		`"2013-10-07T04:23:19.12-04:00"`,
		"3.141592653589793", "false",
		"2456572.849526851851852", "[]",
	}
	for rows.Next() {
		var got json.RawMessage
		err = rows.Scan(sqlite3.JSON(&got))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want[0] {
			t.Errorf("got %q, want %q", got, want[0])
		}
		want = want[1:]
	}
}
