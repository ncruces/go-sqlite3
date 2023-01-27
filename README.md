# Go bindings to SQLite using Wazero

[![Go Reference](https://pkg.go.dev/badge/image)](https://pkg.go.dev/github.com/ncruces/go-sqlite3)
[![Go Report](https://goreportcard.com/badge/github.com/ncruces/go-sqlite3)](https://goreportcard.com/report/github.com/ncruces/go-sqlite3)
[![Go Coverage](https://github.com/ncruces/go-sqlite3/wiki/coverage.svg)](https://raw.githack.com/wiki/ncruces/go-sqlite3/coverage.html)

⚠️ CAUTION ⚠️

This is still very much a WIP.\
DO NOT USE this with data you care about.

Roadmap:
- [x] build SQLite using `zig cc --target=wasm32-wasi`
- [x] `:memory:` databases
- [x] port [`test_demovfs.c`](https://www.sqlite.org/src/doc/trunk/src/test_demovfs.c) to Go
  - branch [`wasi`](https://github.com/ncruces/go-sqlite3/tree/wasi) uses `test_demovfs.c` directly
- [x] come up with a simple, nice API, enough for simple queries
- [ ] file locking, compatible with SQLite on Windows/Unix
- [ ] ~shared-memory, compatible with SQLite on Windows/Unix~