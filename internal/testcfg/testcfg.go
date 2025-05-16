package testcfg

import (
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"

	"github.com/ncruces/go-sqlite3"
)

// notest

func init() {
	// This is the configuration used by most tests.
	sqlite3.RuntimeConfig = wazero.NewRuntimeConfigInterpreter().WithMemoryLimitPages(512)
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
