# Go bindings to SQLite using Wazero

⚠️ DO NOT USE with data you care about. ⚠️

This is very much a WIP.

Roadmap:
- [x] build SQLite using `zig cc --target=wasm32-wasi`
- [x] `:memory:` databases
- [x] use [`test_demovfs.c`](sqlite3/test_demovfs.c) for file databases
- [ ] come up with a simple, nice API:
  - idiomatic Go, close to the C SQLite API:
    - [`github.com/bvinc/go-sqlite-lite`](https://github.com/bvinc/go-sqlite-lite)
    - [`github.com/crawshaw/sqlite`](https://github.com/crawshaw/sqlite)
    - [`github.com/zombiezen/go-sqlite`](https://github.com/zombiezen/go-sqlite)
    - [`github.com/cvilsmeier/sqinn-go`](https://github.com/cvilsmeier/sqinn-go)
  - [`database/sql`](https://pkg.go.dev/database/sql) drivers can come later, if ever
- [ ] implement own VFS to sidestep WASI syscalls:
  - locking will be a pain point:
    - WASM has no threads
    - concurrency is achieved by instantiating the module repeatedly
    - file access needs to be synchronized, both in-process and out-of-process
    - out-of-process should be compatible with SQLite on Windows/Unix
- [ ] benchmark to see if this is usefull at all:
  - [`github.com/cvilsmeier/sqinn-go`](https://github.com/cvilsmeier/sqinn-go-bench)
  - [`github.com/bvinc/go-sqlite-lite`](https://github.com/bvinc/go-sqlite-lite)
  - [`github.com/mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3)
  - [`modernc.org/sqlite`](https://modernc.org/sqlite)

Random TODO list:
- create a Go VFS that's enough to use `:memory:` databases without WASI;
- expand that VFS to wrap an `io.ReaderAt`;
- optimize small allocations that last a single function call;
- …