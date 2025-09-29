package zorder_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/zorder"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/vfs/memdb"
)

func Test_zorder(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, zorder.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got int64
	err = db.QueryRow(`SELECT zorder(2, 3)`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 14 {
		t.Errorf("got %d, want 14", got)
	}

	err = db.QueryRow(`SELECT zorder(4, 5)`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 50 {
		t.Errorf("got %d, want 14", got)
	}

	var check bool
	err = db.QueryRow(`SELECT zorder(3, 4) BETWEEN zorder(2, 3) AND zorder(4, 5)`).Scan(&check)
	if err != nil {
		t.Fatal(err)
	}
	if !check {
		t.Error("want true")
	}

	err = db.QueryRow(`SELECT zorder(2, 2) NOT BETWEEN zorder(2, 3) AND zorder(4, 5)`).Scan(&check)
	if err != nil {
		t.Fatal(err)
	}
	if !check {
		t.Error("want true")
	}
}

func Test_unzorder(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, zorder.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got int64
	err = db.QueryRow(`SELECT unzorder(zorder(3, 4), 2, 0)`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Errorf("got %d, want 3", got)
	}

	err = db.QueryRow(`SELECT unzorder(zorder(3, 4), 2, 1)`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != 4 {
		t.Errorf("got %d, want 4", got)
	}
}

func Test_zorder_error(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, zorder.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got int64
	err = db.QueryRow(`SELECT zorder(1, 2, 3, 100000)`).Scan(&got)
	if err == nil {
		t.Error("want error")
	}

	var buf strings.Builder
	buf.WriteString("SELECT zorder(0")
	for i := 1; i < 25; i++ {
		buf.WriteByte(',')
		buf.WriteString(strconv.Itoa(0))
	}
	buf.WriteByte(')')
	err = db.QueryRow(buf.String()).Scan(&got)
	if err == nil {
		t.Error("want error")
	}
}

func Test_unzorder_error(t *testing.T) {
	t.Parallel()
	dsn := memdb.TestDB(t)

	db, err := driver.Open(dsn, zorder.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got int64
	err = db.QueryRow(`SELECT unzorder(-1, 2, 0)`).Scan(&got)
	if err == nil {
		t.Error("want error")
	}

	err = db.QueryRow(`SELECT unzorder(0, 2, 2)`).Scan(&got)
	if err == nil {
		t.Error("want error")
	}

	err = db.QueryRow(`SELECT unzorder(0, 25, 2)`).Scan(&got)
	if err == nil {
		t.Error("want error")
	}
}
