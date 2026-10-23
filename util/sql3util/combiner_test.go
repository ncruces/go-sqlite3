package sql3util_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/util/sql3util"
)

func ExampleWindowFromCombiner() {
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		return c.CreateWindowFunction("bit_or", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS,
			sql3util.WindowFromCombiner(100, func() *bitOr { return &bitOr{} }))
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE flags (mask INTEGER);
		INSERT INTO flags VALUES (1), (2), (4), (4), (8);
	`)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query(`SELECT bit_or(mask) OVER (ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) FROM flags`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var mask int64
		if err := rows.Scan(&mask); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%04b\n", mask)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// 0011
	// 0111
	// 0110
	// 1100
	// 1100
}

type bitOr struct {
	val int64
	ok  bool
}

func (b *bitOr) Step(ctx sqlite3.Context, arg ...sqlite3.Value) {
	if arg[0].NumericType() <= sqlite3.FLOAT {
		b.val |= arg[0].Int64()
		b.ok = true
	}
}

func (b *bitOr) Combine(other *bitOr) {
	if other.ok {
		b.val |= other.val
		b.ok = true
	}
}

func (b *bitOr) Value(ctx sqlite3.Context) {
	if b.ok {
		ctx.ResultInt64(b.val)
	}
}

func TestWindowFromCombiner_overflow(t *testing.T) {
	t.Parallel()

	ctx := testcfg.Context(t)
	db, err := driver.Open(":memory:", func(c *sqlite3.Conn) error {
		return c.CreateWindowFunction("bit_or", 1, sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS,
			sql3util.WindowFromCombiner(100, func() *bitOr { return &bitOr{} }))
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Standard aggregate exceeding maxWindow degrades gracefully.
	var got int64
	err = db.QueryRowContext(ctx, `SELECT bit_or(value) FROM generate_series(1, 113)`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 127 {
		t.Errorf("got %b, want 127", got)
	}

	// Sliding window exceeding maxWindow errors on subsequent Inverse.
	_, err = db.ExecContext(ctx, `SELECT bit_or(value) OVER (ROWS 100 PRECEDING) FROM generate_series(1, 113)`)
	if err == nil {
		t.Error("want error")
	}
}
