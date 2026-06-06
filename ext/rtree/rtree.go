// Package rtree provides the rtree and geopoly virtual tables.
//
// https://sqlite.org/rtree.html
// https://sqlite.org/geopoly.html
package rtree

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3-wasm/v3/rtree"
)

// Register registers the rtree and geopoly virtual tables.
func Register(db *sqlite3.Conn) error {
	return sqlite3.ExtensionInit(db, rtree.New, rtree.DylinkInfo)
}
