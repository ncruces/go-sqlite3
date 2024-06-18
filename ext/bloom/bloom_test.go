package bloom_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/bloom"
)

//go:embed testdata/bloom.db
var testDB []byte

func TestRegister(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "bloom.db")
	err := os.WriteFile(tmp, testDB, 0666)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sqlite3.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	bloom.Register(db)

	db.Exec(`SELECT COUNT(*) FROM plants('apple')`)
}
