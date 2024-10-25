package memdb

import (
	_ "embed"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

//go:embed testdata/wal.db
var walDB []byte

func Test_wal(t *testing.T) {
	t.Parallel()

	Create("test.db", walDB)

	db, err := sqlite3.Open("file:/test.db?vfs=memdb")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`CREATE TABLE users (id INT, name VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}
}
