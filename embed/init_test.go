package embed

import (
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test_init(t *testing.T) {
	db, err := driver.Open("file:/test.db?vfs=memdb")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var version string
	err = db.QueryRow(`SELECT sqlite_version()`).Scan(&version)
	if err != nil {
		t.Fatal(err)
	}
	if version != "3.46.1" {
		t.Error(version)
	}
}
