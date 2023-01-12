package embed

import (
	_ "embed"

	"github.com/ncruces/go-sqlite3"
)

//go:embed sqlite3.wasm
var binary []byte

func init() {
	sqlite3.Binary = binary
}
