package tests

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestCreateModule_delete(t *testing.T) {
	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = sqlite3.CreateModule[sqlite3.VTab](db, "generate_series", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
}
