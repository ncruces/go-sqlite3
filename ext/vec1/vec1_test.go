package vec1_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/vec1"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test(t *testing.T) {
	dsn := memdb.TestDB(t)

	ctx := testcfg.Context(t)
	db, err := driver.Open(dsn, vec1.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var n float64
	err = db.QueryRowContext(ctx, `SELECT vec1_config('nthread')`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("got %v, want 1", n)
	}

	err = db.QueryRowContext(ctx, `SELECT vec1_config('nprobe')`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0.05 {
		t.Errorf("got %v, want 0.05", n)
	}

	_, err = db.ExecContext(ctx, `
		CREATE VIRTUAL TABLE v1 USING vec1;
		INSERT INTO v1(cmd, vector) VALUES('rebuild', '{index:"flat"}');
		INSERT INTO v1(rowid, vector) VALUES(
			1, vec1_from_json('[1,2,3,4,5,6,7,8]')
		);
		PRAGMA integrity_check;
		DROP TABLE v1;
	`)
	if err != nil {
		t.Fatal(err)
	}
}
