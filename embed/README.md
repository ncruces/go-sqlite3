# Embeddable WASM build of SQLite

This folder includes an embeddable WASM build of SQLite 3.41.0 for use with
[`github.com/ncruces/go-sqlite3`](https://pkg.go.dev/github.com/ncruces/go-sqlite3).

The following optional features are compiled in:
- math functions
- FTS3/4/5
- JSON
- R*Tree
- GeoPoly

See the [configuration options](../sqlite3/sqlite_cfg.h).

Built using [`zig`](https://ziglang.org/) version 0.10.1.