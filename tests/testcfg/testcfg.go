package testcfg

import (
	"github.com/ncruces/go-sqlite3"
	"github.com/tetratelabs/wazero"
)

func init() {
	sqlite3.RuntimeConfig = wazero.NewRuntimeConfig().WithMemoryLimitPages(1024)
}
