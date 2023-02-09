package tests

import (
	"errors"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestDir(t *testing.T) {
	_, err := sqlite3.Open(".")
	if err == nil {
		t.Fatal("want error")
	}
	var serr *sqlite3.Error
	if !errors.As(err, &serr) {
		t.Fatal("want sqlite3.Error")
	}
	if serr.Code() != sqlite3.CANTOPEN {
		t.Error("want sqlite3.CANTOPEN")
	}
	if got := err.Error(); got != "sqlite3: unable to open database file" {
		t.Error("got message: ", got)
	}
}
