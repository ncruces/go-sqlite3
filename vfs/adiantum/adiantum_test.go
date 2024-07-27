package adiantum_test

import (
	_ "embed"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/ncruces/go-sqlite3/vfs/readervfs"
)

//go:embed testdata/test.db
var testDB string

func Test_fileformat(t *testing.T) {
	readervfs.Create("test.db", ioutil.NewSizeReaderAt(strings.NewReader(testDB)))
	adiantum.Register("radiantum", vfs.Find("reader"), nil)

	db, err := driver.Open("file:test.db?vfs=radiantum")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`PRAGMA textkey='correct+horse+battery+staple'`)
	if err != nil {
		t.Fatal(err)
	}

	var version uint32
	err = db.QueryRow(`PRAGMA user_version`).Scan(&version)
	if err != nil {
		t.Fatal(err)
	}
	if version != 0xBADDB {
		t.Error(version)
	}
}

func Benchmark_nokey(b *testing.B) {
	tmp := filepath.Join(b.TempDir(), "test.db")
	sqlite3.Initialize()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		db, err := sqlite3.Open("file:" + filepath.ToSlash(tmp) + "?nolock=1")
		if err != nil {
			b.Fatal(err)
		}
		db.Close()
	}
}
func Benchmark_hexkey(b *testing.B) {
	tmp := filepath.Join(b.TempDir(), "test.db")
	sqlite3.Initialize()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		db, err := sqlite3.Open("file:" + filepath.ToSlash(tmp) + "?nolock=1" +
			"&vfs=adiantum&hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
		if err != nil {
			b.Fatal(err)
		}
		db.Close()
	}
}

func Benchmark_textkey(b *testing.B) {
	tmp := filepath.Join(b.TempDir(), "test.db")
	sqlite3.Initialize()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		db, err := sqlite3.Open("file:" + filepath.ToSlash(tmp) + "?nolock=1" +
			"&vfs=adiantum&textkey=correct+horse+battery+staple")
		if err != nil {
			b.Fatal(err)
		}
		db.Close()
	}
}
