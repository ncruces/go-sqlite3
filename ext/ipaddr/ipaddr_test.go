package ipaddr_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/ipaddr"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func TestRegister(t *testing.T) {
	t.Parallel()
	tmp := memdb.TestDB(t)

	db, err := driver.Open(tmp, ipaddr.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got string

	err = db.QueryRow(`SELECT ipcontains('192.168.1.0/24', '192.168.1.5')`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1" {
		t.Fatalf("got %s", got)
	}

	err = db.QueryRow(`SELECT ipfamily('::1')`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != "6" {
		t.Fatalf("got %s", got)
	}

	err = db.QueryRow(`SELECT ipfamily('192.168.1.5/24')`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != "4" {
		t.Fatalf("got %s", got)
	}

	err = db.QueryRow(`SELECT iphost('192.168.1.5/24')`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != "192.168.1.5" {
		t.Fatalf("got %s", got)
	}

	err = db.QueryRow(`SELECT ipmasklen('192.168.1.5/24')`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != "24" {
		t.Fatalf("got %s", got)
	}

	err = db.QueryRow(`SELECT ipnetwork('192.168.1.5/24')`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != "192.168.1.0/24" {
		t.Fatalf("got %s", got)
	}
}
