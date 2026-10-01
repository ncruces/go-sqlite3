//go:build ignore

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/ext/vec1"
)

func main() {
	basePath := flag.String("base", "siftsmall_base.fvecs", "path to base .fvecs file")
	queryPath := flag.String("query", "siftsmall_query.fvecs", "path to query .fvecs file")
	outPath := flag.String("out", "siftsmall.db", "output database path")
	limit := flag.Int("limit", 2000, "maximum number of base vectors to import (0 for all)")
	flag.Parse()

	_ = os.Remove(*outPath)

	db, err := sqlite3.Open(*outPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := vec1.Register(db); err != nil {
		log.Fatalf("failed to register vec1: %v", err)
	}

	// 1. Create vec1 table and configure flat index
	err = db.Exec(`
		CREATE VIRTUAL TABLE siftsmall USING vec1(vector);
		INSERT INTO siftsmall(cmd, arg) VALUES('rebuild', '{"index":"flat","distance":"l2"}');
		CREATE TABLE queries (
			id INTEGER PRIMARY KEY,
			vector BLOB
		);
	`)
	if err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	// 2. Import base vectors
	fmt.Printf("Importing base vectors from %s (limit: %d)...\n", *basePath, *limit)
	nBase, err := importFVecs(db, *basePath, *limit, "INSERT INTO siftsmall(rowid, vector) VALUES(?, ?)")
	if err != nil {
		log.Fatalf("failed to import base vectors: %v", err)
	}
	fmt.Printf("Imported %d base vectors.\n", nBase)

	// 3. Import query vectors (all 100 queries)
	fmt.Printf("Importing query vectors from %s...\n", *queryPath)
	nQueries, err := importFVecs(db, *queryPath, 0, "INSERT INTO queries(id, vector) VALUES(?, ?)")
	if err != nil {
		log.Fatalf("failed to import query vectors: %v", err)
	}
	fmt.Printf("Imported %d query vectors.\n", nQueries)

	// Rebuild and vacuum to compact
	if err := db.Exec(`INSERT INTO siftsmall(cmd, arg) VALUES('rebuild', '{"index":"flat","distance":"l2"}'); VACUUM;`); err != nil {
		log.Fatalf("rebuild/vacuum failed: %v", err)
	}
	fmt.Println("Database built successfully.")
}

func importFVecs(db *sqlite3.Conn, path string, limit int, query string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	stmt, _, err := db.Prepare(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	tx := db.Begin()
	defer tx.End(&err)

	count := 0
	dimBuf := make([]byte, 4)

	for {
		if limit > 0 && count >= limit {
			break
		}
		_, err := io.ReadFull(f, dimBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}

		dim := int(binary.LittleEndian.Uint32(dimBuf))
		vecBytes := make([]byte, dim*4)
		if _, err := io.ReadFull(f, vecBytes); err != nil {
			return count, err
		}

		count++
		if err := stmt.BindInt64(1, int64(count)); err != nil {
			return count, err
		}
		if err := stmt.BindBlob(2, vecBytes); err != nil {
			return count, err
		}
		if err := stmt.Exec(); err != nil {
			return count, err
		}
	}

	return count, nil
}
