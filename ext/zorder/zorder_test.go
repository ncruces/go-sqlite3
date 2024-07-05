package zorder_test

import (
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/zorder"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister_zorder(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", zorder.Register)
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

func TestRegister_unzorder(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", zorder.Register)
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

func TestRegister_error(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", zorder.Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var got int64
	err = db.QueryRow(`SELECT zorder(1, 2, 3, 100000)`).Scan(&got)
	if err == nil {
		t.Error("want error")
	}
}
