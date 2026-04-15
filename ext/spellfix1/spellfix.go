// Package spellfix1 provides the spellfix1 virtual table.
//
// https://sqlite.org/spellfix1.html
package spellfix1

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3-wasm/spellfix"
)

// Register registers the spellfix1 virtual table.
func Register(db *sqlite3.Conn) error {
	return sqlite3.ExtensionInit(db, spellfix.New, spellfix.DylinkInfo)
}
