package regexp

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister(t *testing.T) {
	db, err := driver.Open(":memory:", func(conn *sqlite3.Conn) error {
		Register(conn)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
}
