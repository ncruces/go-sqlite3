package regexp

import (
	"testing"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []struct {
		test string
		want string
	}{
		{`'Hello' REGEXP 'elo'`, "0"},
		{`'Hello' REGEXP 'ell'`, "1"},
		{`'Hello' REGEXP 'el.'`, "1"},
		{`regexp_like('Hello', 'elo')`, "0"},
		{`regexp_like('Hello', 'ell')`, "1"},
		{`regexp_like('Hello', 'el.')`, "1"},
		{`regexp_substr('Hello', 'el.')`, "ell"},
		{`regexp_replace('Hello', 'llo', 'll')`, "Hell"},
	}

	for _, tt := range tests {
		var got string
		err := db.QueryRow(`SELECT ` + tt.test).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestRegister_errors(t *testing.T) {
	t.Parallel()

	db, err := driver.Open(":memory:", Register)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tests := []string{
		`'' REGEXP ?`,
		`regexp_like('', ?)`,
		`regexp_substr('', ?)`,
		`regexp_replace('', ?, '')`,
	}

	for _, tt := range tests {
		err := db.QueryRow(`SELECT `+tt, `\`).Scan(nil)
		if err == nil {
			t.Fatal("want error")
		}
	}
}
