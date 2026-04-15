// Package vec1 provides the vec1 vector extension.
//
// https://sqlite.org/vec1
package vec1

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3-wasm/vec1"
)

// Register registers the vec1 vector extension.
func Register(db *sqlite3.Conn) error {
	return sqlite3.ExtensionInit(db, vec1.New, vec1.DylinkInfo)
}
