// Package fts5 provides the fts5 extension.
//
// https://sqlite.org/fts5.html
package fts5

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3-wasm/v3/fts5"
)

// Register registers the fts5 extension.
func Register(db *sqlite3.Conn) error {
	return sqlite3.ExtensionInit(db, fts5.New, fts5.DylinkInfo)
}
