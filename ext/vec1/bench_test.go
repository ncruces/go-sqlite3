package vec1_test

import (
	"bytes"
	"compress/bzip2"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/vec1"
	"github.com/ncruces/go-sqlite3/util/ioutil"
	"github.com/ncruces/go-sqlite3/vfs/readervfs"
)

var (
	//go:embed testdata/siftsmall.db.bz2
	siftsmallBZ2 string
	siftsmall    []byte
)

func TestMain(m *testing.M) {
	var err error
	siftsmall, err = io.ReadAll(bzip2.NewReader(strings.NewReader(siftsmallBZ2)))
	if err != nil {
		panic(err)
	}
	readervfs.Create("siftsmall.db", ioutil.NewSizeReaderAt(bytes.NewReader(siftsmall)))
	os.Exit(m.Run())
}

// loadBenchmarkDB opens the in-memory database using readervfs to isolate
// CPU performance from filesystem I/O.
func loadBenchmarkDB(tb testing.TB) (*sqlite3.Conn, [][]byte) {
	tb.Helper()

	db, err := sqlite3.Open("file:siftsmall.db?vfs=reader")
	if err != nil {
		tb.Fatalf("open reader db: %v", err)
	}
	if err := vec1.Register(db); err != nil {
		tb.Fatalf("register vec1: %v", err)
	}

	// Preload query vectors into memory
	stmt, _, err := db.Prepare(`SELECT vector FROM queries ORDER BY id`)
	if err != nil {
		tb.Fatalf("prepare queries: %v", err)
	}
	defer stmt.Close()

	var queries [][]byte
	for stmt.Step() {
		queries = append(queries, stmt.ColumnBlob(0, nil))
	}
	if err := stmt.Err(); err != nil {
		tb.Fatalf("fetching queries: %v", err)
	}

	return db, queries
}

func TestValidation(t *testing.T) {
	db, queries := loadBenchmarkDB(t)
	defer db.Close()

	if len(queries) == 0 {
		t.Fatal("no queries found")
	}

	stmt, _, err := db.Prepare(`SELECT rowid, distance FROM siftsmall(?, '{"k": 5}')`)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()

	// Validate query 0 returns 5 ascending matches
	if err := stmt.BindBlob(1, queries[0]); err != nil {
		t.Fatal(err)
	}

	var count int
	var lastDist float64
	for stmt.Step() {
		rowid := stmt.ColumnInt64(0)
		dist := stmt.ColumnFloat(1)
		if count > 0 && dist < lastDist {
			t.Errorf("distances not ascending: rowid=%d dist=%f last=%f", rowid, dist, lastDist)
		}
		lastDist = dist
		count++
	}
	if err := stmt.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Errorf("expected 5 matches, got %d", count)
	}
}

func BenchmarkKNN(b *testing.B) {
	for _, k := range []int{1, 10, 50} {
		b.Run(fmt.Sprintf("k=%d", k), func(b *testing.B) {
			db, queries := loadBenchmarkDB(b)
			defer db.Close()

			queryJSON := fmt.Sprintf(`{"k": %d}`, k)
			stmt, _, err := db.Prepare(`SELECT rowid, distance FROM siftsmall(?, ?)`)
			if err != nil {
				b.Fatal(err)
			}
			defer stmt.Close()

			if err := stmt.BindText(2, queryJSON); err != nil {
				b.Fatal(err)
			}

			numQueries := len(queries)
			b.ResetTimer()

			for i := range b.N {
				q := queries[i%numQueries]
				if err := stmt.BindBlob(1, q); err != nil {
					b.Fatal(err)
				}
				for stmt.Step() {
					// Consume result row (pure CPU)
					_ = stmt.ColumnInt64(0)
					_ = stmt.ColumnFloat(1)
				}
				if err := stmt.Reset(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
