package unicode

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec := func(fn string) string {
		stmt, _, err := db.Prepare(`SELECT ` + fn)
		if err != nil {
			t.Fatal(err)
		}
		defer stmt.Close()

		if stmt.Step() {
			return stmt.ColumnText(0)
		}
		t.Fatal(stmt.Err())
		return ""
	}

	Register(db)

	tests := []struct {
		test string
		want string
	}{
		{`upper('hello')`, "HELLO"},
		{`lower('HELLO')`, "hello"},
		{`upper('привет')`, "ПРИВЕТ"},
		{`lower('ПРИВЕТ')`, "привет"},
		{`upper('istanbul')`, "ISTANBUL"},
		{`upper('istanbul', 'tr-TR')`, "İSTANBUL"},
		{`lower('Dünyanın İlk Borsası', 'tr-TR')`, "dünyanın ilk borsası"},
		{`upper('Dünyanın İlk Borsası', 'tr-TR')`, "DÜNYANIN İLK BORSASI"},
		{`initcap('Kad je hladno Marko nosi džemper')`, "Kad Je Hladno Marko Nosi Džemper"},
		{`initcap('Kad je hladno Marko nosi džemper', 'hr-HR')`, "Kad Je Hladno Marko Nosi Džemper"},
		{`normalize(X'61cc88')`, "ä"},
		{`normalize(X'61cc88', 'NFC' )`, "ä"},
		{`normalize(X'61cc88', 'NFKC')`, "ä"},
		{`normalize('ä', 'NFD' )`, "\x61\xcc\x88"},
		{`normalize('ä', 'NFKD')`, "\x61\xcc\x88"},
		{`casefold('Maße')`, "masse"},
		{`unaccent('Hôtel')`, "Hotel"},
		{`'Hello' REGEXP 'ell'`, "1"},
		{`'Hello' REGEXP 'el.'`, "1"},
		{`'Hello' LIKE 'hel_'`, "0"},
		{`'Hello' LIKE 'hel%'`, "1"},
		{`'Hello' LIKE 'h_llo'`, "1"},
		{`'Hello' LIKE 'hello'`, "1"},
		{`'Привет' LIKE 'ПРИВЕТ'`, "1"},
		{`'100%' LIKE '100|%' ESCAPE '|'`, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.test, func(t *testing.T) {
			if got := exec(tt.test); got != tt.want {
				t.Errorf("exec(%q) = %q, want %q", tt.test, got, tt.want)
			}
		})
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegister_collation(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	Register(db)

	err = db.Exec(`CREATE TABLE words (word VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO words (word) VALUES ('côte'), ('cote'), ('coter'), ('coté'), ('cotée'), ('côté')`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`SELECT icu_load_collation('fr-FR', 'french')`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT word FROM words ORDER BY word COLLATE french`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	got, want := []string{}, []string{"cote", "coté", "côte", "côté", "cotée", "coter"}

	for stmt.Step() {
		got = append(got, stmt.ColumnText(0))
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegisterCollationsNeeded(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	RegisterCollationsNeeded(db)

	err = db.Exec(`CREATE TABLE words (word VARCHAR(10))`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`INSERT INTO words (word) VALUES ('côte'), ('cote'), ('coter'), ('coté'), ('cotée'), ('côté')`)
	if err != nil {
		t.Fatal(err)
	}

	stmt, _, err := db.Prepare(`SELECT word FROM words ORDER BY word COLLATE fr_FR`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	got, want := []string{}, []string{"cote", "coté", "côte", "côté", "cotée", "coter"}

	for stmt.Step() {
		got = append(got, stmt.ColumnText(0))
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Error("not equal")
	}

	err = stmt.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRegister_error(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	Register(db)

	err = db.Exec(`SELECT upper('hello', 'enUS')`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.ERROR) {
		t.Errorf("got %v, want sqlite3.ERROR", err)
	}

	err = db.Exec(`SELECT lower('hello', 'enUS')`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.ERROR) {
		t.Errorf("got %v, want sqlite3.ERROR", err)
	}

	err = db.Exec(`SELECT normalize('', 'NF')`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.ERROR) {
		t.Errorf("got %v, want sqlite3.ERROR", err)
	}

	err = db.Exec(`SELECT 'hello' REGEXP '\'`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.ERROR) {
		t.Errorf("got %v, want sqlite3.ERROR", err)
	}

	err = db.Exec(`SELECT 'hello' LIKE 'HELLO' ESCAPE '\\'`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.ERROR) {
		t.Errorf("got %v, want sqlite3.ERROR", err)
	}

	err = db.Exec(`SELECT icu_load_collation('enUS', 'error')`)
	if err == nil {
		t.Error("want error")
	}
	if !errors.Is(err, sqlite3.ERROR) {
		t.Errorf("got %v, want sqlite3.ERROR", err)
	}

	err = db.Exec(`SELECT icu_load_collation('enUS', '')`)
	if err != nil {
		t.Error(err)
	}

	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_like2regex(t *testing.T) {
	t.Parallel()

	const prefix = `(?is)\A`
	const sufix = `\z`
	tests := []struct {
		pattern string
		escape  rune
		want    string
	}{
		{`a`, -1, `a`},
		{`a.`, -1, `a\.`},
		{`a%`, -1, `a.*`},
		{`a\`, -1, `a\\`},
		{`a_b`, -1, `a.b`},
		{`a|b`, '|', `ab`},
		{`a|_`, '|', `a_`},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			want := prefix + tt.want + sufix
			if got := like2regex(tt.pattern, tt.escape); got != want {
				t.Errorf("like2regex() = %q, want %q", got, want)
			}
		})
	}
}
