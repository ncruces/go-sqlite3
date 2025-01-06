package testcfg

import (
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// notest

func init() {
	if util.CompilerSupported() {
		sqlite3.RuntimeConfig = wazero.NewRuntimeConfigCompiler()
	} else {
		sqlite3.RuntimeConfig = wazero.NewRuntimeConfigInterpreter()
	}
	sqlite3.RuntimeConfig = sqlite3.RuntimeConfig.WithMemoryLimitPages(512)
	if os.Getenv("CI") != "" {
		path := filepath.Join(os.TempDir(), "wazero")
		if err := os.MkdirAll(path, 0777); err == nil {
			if cache, err := wazero.NewCompilationCacheWithDir(path); err == nil {
				sqlite3.RuntimeConfig = sqlite3.RuntimeConfig.
					WithCompilationCache(cache)
			}
		}
	}
}
