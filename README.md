# Go bindings to SQLite using Wazero

[![Go Reference](https://pkg.go.dev/badge/image)](https://pkg.go.dev/github.com/ncruces/go-sqlite3)
[![Go Report](https://goreportcard.com/badge/github.com/ncruces/go-sqlite3)](https://goreportcard.com/report/github.com/ncruces/go-sqlite3)
[![Go Coverage](https://github.com/ncruces/go-sqlite3/wiki/coverage.svg)](https://github.com/ncruces/go-sqlite3/wiki/Test-coverage-report)

Go module `github.com/ncruces/go-sqlite3` wraps a [WASM](https://webassembly.org/) build of [SQLite](https://sqlite.org/),
and uses [wazero](https://wazero.io/) to provide `cgo`-free SQLite bindings.

- Package [`github.com/ncruces/go-sqlite3`](https://pkg.go.dev/github.com/ncruces/go-sqlite3)
wraps the [C SQLite API](https://www.sqlite.org/cintro.html)
([example usage](https://pkg.go.dev/github.com/ncruces/go-sqlite3#example-package)).
- Package [`github.com/ncruces/go-sqlite3/driver`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/driver)
provides a [`database/sql`](https://pkg.go.dev/database/sql) driver
([example usage](https://pkg.go.dev/github.com/ncruces/go-sqlite3/driver#example-package)).
- Package [`github.com/ncruces/go-sqlite3/embed`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/embed)
embeds a build of SQLite into your application.

### Caveats

#### Write-Ahead Logging

Because WASM does not support shared memory,
[WAL](https://www.sqlite.org/wal.html) support is [limited](https://www.sqlite.org/wal.html#noshm).

To work around this limitation, SQLite is compiled with
[`SQLITE_DEFAULT_LOCKING_MODE=1`](https://www.sqlite.org/compile.html#default_locking_mode),
making `EXCLUSIVE` the default locking mode.
For non-WAL databases, `NORMAL` locking mode can be activated with
[`PRAGMA locking_mode=NORMAL`](https://www.sqlite.org/pragma.html#pragma_locking_mode).

Because connection pooling is incompatible with `EXCLUSIVE` locking mode,
the `database/sql` driver defaults to `NORMAL` locking mode,
and WAL databases are not supported.

#### Open File Description Locks

On Unix, this module uses [OFD locks](https://www.gnu.org/software/libc/manual/html_node/Open-File-Description-Locks.html)
to synchronize access to database files.

POSIX advisory locks, which SQLite uses, are [broken by design](https://www.sqlite.org/src/artifact/90c4fa?ln=1073-1161).
OFD locks are fully compatible with process-associated POSIX advisory locks,
and are supported on Linux, macOS and illumos.
As a work around for other Unixes, you can use [`nolock=1`](https://www.sqlite.org/uri.html).

### Roadmap

- [x] build SQLite using `zig cc --target=wasm32-wasi`
- [x] `:memory:` databases
- [x] port [`test_demovfs.c`](https://www.sqlite.org/src/doc/trunk/src/test_demovfs.c) to Go
  - branch [`wasi`](https://github.com/ncruces/go-sqlite3/tree/wasi) uses `test_demovfs.c` directly
- [x] design a nice API, enough for simple use cases
- [x] provide a simple `database/sql` driver
- [x] file locking, compatible with SQLite on macOS/Linux/Windows
- [ ] advanced SQLite features
  - [x] nested transactions
  - [x] incremental BLOB I/O
  - [ ] online backup
  - [ ] snapshots
  - [ ] session extension
  - [ ] resumable bulk update
  - [ ] shared cache mode
  - [ ] unlock-notify
- [ ] custom SQL functions
- [ ] custom VFSes
  - [ ] read-only VFS, wrapping an [`io.ReaderAt`](https://pkg.go.dev/io#ReaderAt)
  - [ ] in-memory VFS, wrapping a [`bytes.Buffer`](https://pkg.go.dev/bytes#Buffer)
  - [ ] cloud-based VFS, based on [Cloud Backed SQLite](https://sqlite.org/cloudsqlite/doc/trunk/www/index.wiki)
  - [ ] custom VFS API
  
### Alternatives

- [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite)
- [`crawshaw.io/sqlite`](https://pkg.go.dev/crawshaw.io/sqlite)
- [`github.com/mattn/go-sqlite3`](https://pkg.go.dev/github.com/mattn/go-sqlite3)
- [`github.com/zombiezen/go-sqlite`](https://pkg.go.dev/github.com/zombiezen/go-sqlite)