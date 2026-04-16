package tests

import (
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
)

func Test_base64(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		s    string
		want string
	}{
		{"x''", ""},
		{"''", ""},

		{"x'000102030405'", "AAECAwQF"},
		{"x'0001020304'", "AAECAwQ="},
		{"x'00010203'", "AAECAw=="},

		{"'AAECAwQF'", "\x00\x01\x02\x03\x04\x05"},
		{"'AAECAwQ='", "\x00\x01\x02\x03\x04"},
		{"'AAECAw=='", "\x00\x01\x02\x03"},

		{"' AAECAwQF '", "\x00\x01\x02\x03\x04\x05"},
		{"' AAECAwQ'", "\x00\x01\x02\x03\x04"},
		{"' AAECAw '", "\x00\x01\x02\x03"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			stmt, _, err := db.Prepare(`SELECT base64(` + tt.s + `)`)
			if err != nil {
				t.Error(err)
			}
			defer stmt.Close()

			if !stmt.Step() {
				t.Fatal("expected one row")
			}
			if got := stmt.ColumnText(0); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_decimal(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT decimal_add(decimal('0.1'), decimal('0.2')) = decimal('0.3')`)
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal("expected one row")
	}
	if !stmt.ColumnBool(0) {
		t.Error("want true")
	}
}

func Test_uint(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.OpenContext(testcfg.Context(t), ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stmt, _, err := db.Prepare(`SELECT 'z2' < 'z11' COLLATE UINT`)
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()

	if !stmt.Step() {
		t.Fatal("expected one row")
	}
	if !stmt.ColumnBool(0) {
		t.Error("want true")
	}
}
