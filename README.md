# Go bindings to SQLite using Wazero

⚠️ DO NOT USE with data you care about. ⚠️

This is very much a WIP.

Roadmap:
- [x] build SQLite using `zig cc --target=wasm32-wasi`
- [x] `:memory:` databases
- [x] port [`test_demovfs.c`](https://www.sqlite.org/src/doc/trunk/src/test_demovfs.c) to Go
  - branch [`wasi`](https://github.com/ncruces/go-sqlite3/tree/wasi) uses `test_demovfs.c` directly
- [x] come up with a simple, nice API, enough for simple queries
- [ ] file locking, compatible with SQLite on Windows/Unix
- [ ] shared-memory, compatible with SQLite on Windows/Unix

Benchmarks:

```
goos: darwin
goarch: amd64
pkg: github.com/ncruces/go-sqlite3/bench
cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz

BenchmarkCrawshaw-12                   8         677059222 ns/op        104003008 B/op   9000000 allocs/op
BenchmarkWasm-12                       8         702393992 ns/op        178750252 B/op  11000110 allocs/op
BenchmarkWASI-12                       5        1015034369 ns/op        178750009 B/op  11000102 allocs/op

BenchmarkCrawshawFile-12               8         704186415 ns/op        104002593 B/op   8999998 allocs/op
BenchmarkWasmFile-12                   5        1029067495 ns/op        178750070 B/op  11000102 allocs/op
BenchmarkWASIFile-12                   3        2226217997 ns/op        868255072 B/op  16000200 allocs/op
```
