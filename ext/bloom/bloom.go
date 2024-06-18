package bloom

import (
	"fmt"
	"math"
	"strconv"

	"github.com/ncruces/go-sqlite3"
)

func Register(db *sqlite3.Conn) {
	sqlite3.CreateModule(db, "bloom_filter", create, connect)
}

func create(db *sqlite3.Conn, _, schema, table string, arg ...string) (_ *bloom, err error) {
	t := bloom{
		schema:  schema,
		table:   table,
		storage: table + "_storage",
		prob:    0.01,
	}

	nelem := 100
	if len(arg) > 0 {
		nelem, err = strconv.Atoi(arg[0])
		if err != nil {
			return nil, err
		}
		if nelem <= 0 {
			return nil, fmt.Errorf("bloom: number of elements in filter must be positive")
		}
	}

	if len(arg) > 1 {
		t.prob, err = strconv.ParseFloat(arg[1], 64)
		if err != nil {
			return nil, err
		}
		if t.prob <= 0 || t.prob >= 1 {
			return nil, fmt.Errorf("bloom: probability must be in the range (0,1)")
		}
	}

	if len(arg) > 2 {
		t.hashes, err = strconv.Atoi(arg[2])
		if err != nil {
			return nil, err
		}
		if t.hashes <= 0 {
			return nil, fmt.Errorf("bloom: number of hash functions must be positive")
		}
	} else {
		t.hashes = int(math.Round(-math.Log2(t.prob)))
	}

	t.nfilter = computeBytes(nelem, t.prob)

	err = db.Exec(fmt.Sprintf(
		`CREATE TABLE %s.%s (data BLOB, p REAL, n INTEGER, m INTEGER, k INTEGER)`,
		sqlite3.QuoteIdentifier(t.schema), sqlite3.QuoteIdentifier(t.storage)))
	if err != nil {
		return nil, err
	}

	err = db.Exec(fmt.Sprintf(
		`INSERT INTO %s.%s (rowid, data, p, n, m, k)
		 VALUES (1, zeroblob(%d), %f, %d, %d, %d)`,
		sqlite3.QuoteIdentifier(t.schema), sqlite3.QuoteIdentifier(t.storage),
		t.nfilter, t.prob, nelem, t.nfilter*8, t.hashes))
	if err != nil {
		return nil, err
	}

	err = db.DeclareVTab(
		`CREATE TABLE x(present, word HIDDEN NOT NULL PRIMARY KEY) WITHOUT ROWID`)
	if err != nil {
		db.Exec(fmt.Sprintf(`DROP TABLE %s.%s`,
			sqlite3.QuoteIdentifier(schema), sqlite3.QuoteIdentifier(table+"_storage")))
		return nil, err
	}
	return &t, nil
}

func connect(db *sqlite3.Conn, _, schema, table string, arg ...string) (_ *bloom, err error) {
	t := bloom{
		schema:  schema,
		table:   table,
		storage: table + "_storage",
	}

	err = db.DeclareVTab(
		`CREATE TABLE x(present, word HIDDEN NOT NULL PRIMARY KEY) WITHOUT ROWID`)
	if err != nil {
		return nil, err
	}

	load, _, err := db.Prepare(fmt.Sprintf(
		`SELECT m/8, p, k FROM %s.%s WHERE rowid = 1`,
		sqlite3.QuoteIdentifier(t.schema), sqlite3.QuoteIdentifier(t.storage)))
	if err != nil {
		return nil, err
	}
	defer load.Close()

	if !load.Step() {
		if err = load.Err(); err == nil {
			err = sqlite3.CORRUPT_VTAB
		}
		return nil, err
	}

	t.nfilter = load.ColumnInt64(0)
	t.prob = load.ColumnFloat(1)
	t.hashes = load.ColumnInt(2)
	return &t, nil
}

type bloom struct {
	schema  string
	table   string
	storage string
	prob    float64
	nfilter int64
	hashes  int
}

func (bloom) BestIndex(idx *sqlite3.IndexInfo) error {
	return nil
}

func (bloom) Open() (sqlite3.VTabCursor, error) {
	return nil, nil
}

func computeBytes(n int, p float64) int64 {
	bits := math.Ceil(-((float64(n) * math.Log(p)) / (math.Ln2 * math.Ln2)))
	quo := int64(bits) / 8
	rem := int64(bits) % 8
	if rem != 0 {
		quo += 1
	}
	return quo
}
