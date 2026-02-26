package mvcc

import (
	_ "embed"
	"testing"

	"github.com/ncruces/go-sqlite3"
)

//go:embed testdata/wal.db
var walDB string

func Test_wal(t *testing.T) {
	t.Parallel()
	dsn := TestDB(t, NewSnapshot(walDB))

	db, err := sqlite3.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE users (id INT, name VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}
}
