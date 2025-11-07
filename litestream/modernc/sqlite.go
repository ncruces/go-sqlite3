// Package sqlite provides a shim that allows Litestream to work with the ncruces SQLite driver.
package sqlite

import (
	"database/sql"
	"slices"

	"github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func init() {
	if !slices.Contains(sql.Drivers(), "sqlite") {
		sql.Register("sqlite", &driver.SQLite{})
	}
}

type FileControl interface {
	FileControlPersistWAL(string, int) (int, error)
}
