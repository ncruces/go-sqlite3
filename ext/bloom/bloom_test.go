package bloom_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/bloom"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
)

func TestMain(m *testing.M) {
	sqlite3.AutoExtension(bloom.Register)
	os.Exit(m.Run())
}

func TestRegister(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Exec(`
		CREATE VIRTUAL TABLE sports_cars USING bloom_filter();
		INSERT INTO sports_cars VALUES ('ferrari'), ('lamborghini'), ('alfa romeo')
	`)
	if err != nil {
		t.Fatal(err)
	}

	query, _, err := db.Prepare(`SELECT COUNT(*) FROM sports_cars(?)`)
	if err != nil {
		t.Fatal(err)
	}

	err = query.BindText(1, "ferrari")
	if err != nil {
		t.Fatal(err)
	}
	if !query.Step() {
		t.Error("no rows")
	}
	if !query.ColumnBool(0) {
		t.Error("want true")
	}
	err = query.Reset()
	if err != nil {
		t.Fatal(err)
	}

	err = query.BindText(1, "bmw")
	if err != nil {
		t.Fatal(err)
	}
	if !query.Step() {
		t.Error("no rows")
	}
	if query.ColumnBool(0) {
		t.Error("want false")
	}
	err = query.Close()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`DELETE FROM sports_cars WHERE word = 'lamborghini'`)
	if err == nil {
		t.Error("want error")
	}

	err = db.Exec(`UPDATE sports_cars SET word = 'ferrari' WHERE word = 'lamborghini'`)
	if err == nil {
		t.Error("want error")
	}

	err = db.Exec(`ALTER TABLE sports_cars RENAME TO fast_cars`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`DROP TABLE fast_cars`)
	if err != nil {
		t.Fatal(err)
	}
}

//go:embed testdata/bloom.db
var testDB []byte

func Test_compatible(t *testing.T) {
	t.Parallel()

	tmp := filepath.Join(t.TempDir(), "bloom.db")
	err := os.WriteFile(tmp, testDB, 0666)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sqlite3.Open("file:" + filepath.ToSlash(tmp) + "?nolock=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	query, _, err := db.Prepare(`SELECT COUNT(*) FROM plants(?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer query.Close()

	err = query.BindText(1, "apple")
	if err != nil {
		t.Fatal(err)
	}
	if !query.Step() {
		t.Error("no rows")
	}
	if !query.ColumnBool(0) {
		t.Error("want true")
	}
	err = query.Reset()
	if err != nil {
		t.Fatal(err)
	}

	err = query.BindText(1, "lemon")
	if err != nil {
		t.Fatal(err)
	}
	if !query.Step() {
		t.Error("no rows")
	}
	if query.ColumnBool(0) {
		t.Error("want false")
	}
	err = query.Reset()
	if err != nil {
		t.Fatal(err)
	}

	err = db.Exec(`PRAGMA integrity_check`)
	if err != nil {
		t.Error(err)
	}

	err = db.Exec(`PRAGMA quick_check`)
	if err != nil {
		t.Error(err)
	}
}

func Test_errors(t *testing.T) {
	t.Parallel()

	db, err := sqlite3.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	bloom.Register(db)

	err = db.Exec(`CREATE VIRTUAL TABLE sports_cars USING bloom_filter(0)`)
	if err == nil {
		t.Error("want error")
	}
	err = db.Exec(`CREATE VIRTUAL TABLE sports_cars USING bloom_filter('a')`)
	if err == nil {
		t.Error("want error")
	}

	err = db.Exec(`CREATE VIRTUAL TABLE sports_cars USING bloom_filter(20, 2)`)
	if err == nil {
		t.Error("want error")
	}
	err = db.Exec(`CREATE VIRTUAL TABLE sports_cars USING bloom_filter(20, 'a')`)
	if err == nil {
		t.Error("want error")
	}

	err = db.Exec(`CREATE VIRTUAL TABLE sports_cars USING bloom_filter(20, 0.9, 0)`)
	if err == nil {
		t.Error("want error")
	}
	err = db.Exec(`CREATE VIRTUAL TABLE sports_cars USING bloom_filter(20, 0.9, 'a')`)
	if err == nil {
		t.Error("want error")
	}
}
