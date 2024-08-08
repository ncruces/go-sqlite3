// Package bcw2 embeds SQLite into your application.
//
// Importing package bcw2 initializes the [sqlite3.Binary] variable
// with a build of SQLite that includes the [BEGIN CONCURRENT] and [Wal2] patches:
//
//	import _ "github.com/ncruces/go-sqlite3/embed/bcw2"
//
// [BEGIN CONCURRENT]: https://sqlite.org/src/doc/begin-concurrent/doc/begin_concurrent.md
// [Wal2]: https://www.sqlite.org/cgi/src/doc/wal2/doc/wal2.md
package bcw2

import (
	_ "embed"

	"github.com/ncruces/go-sqlite3"
)

//go:embed bcw2.wasm
var binary []byte

func init() {
	sqlite3.Binary = binary
}
