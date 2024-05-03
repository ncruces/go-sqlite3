package adiantum_test

import (
	"path/filepath"
	"testing"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"
)

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
