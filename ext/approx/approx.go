// Package approx provides approximate aggregate functions.
//
// Provided functions:
//   - approx_count_distinct: approximate count of distinct elements using HyperLogLog.
package approx

import "github.com/ncruces/go-sqlite3"

func Register(db *sqlite3.Conn) error {
	return db.CreateWindowFunction("approx_count_distinct", 1,
		sqlite3.DETERMINISTIC|sqlite3.INNOCUOUS,
		newHLL())
}
