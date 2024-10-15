# Go SQLite Extensions

This folder collects optional SQLite extensions
you can load into your database connections.

### Extensions

- [`github.com/ncruces/go-sqlite3/ext/array`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/array)
  provides the [`array`](https://sqlite.org/carray.html) table-valued function.
- [`github.com/ncruces/go-sqlite3/ext/blobio`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/blobio)
  simplifies [incremental BLOB I/O](https://sqlite.org/c3ref/blob_open.html).
- [`github.com/ncruces/go-sqlite3/ext/bloom`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/bloom)
  provides a [Bloom filter](https://github.com/nalgeon/sqlean/issues/27#issuecomment-1002267134) virtual table.
- [`github.com/ncruces/go-sqlite3/ext/closure`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/closure)
  provides a transitive closure virtual table.
- [`github.com/ncruces/go-sqlite3/ext/csv`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/csv)
  reads [comma-separated values](https://sqlite.org/csv.html).
- [`github.com/ncruces/go-sqlite3/ext/fileio`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/fileio)
  reads, writes and lists files.
- [`github.com/ncruces/go-sqlite3/ext/hash`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/hash)
  provides cryptographic hash functions.
- [`github.com/ncruces/go-sqlite3/ext/lines`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/lines)
  reads data [line-by-line](https://github.com/asg017/sqlite-lines).
- [`github.com/ncruces/go-sqlite3/ext/pivot`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/pivot)
  creates [pivot tables](https://github.com/jakethaw/pivot_vtab).
- [`github.com/ncruces/go-sqlite3/ext/regexp`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/regexp)
  provides regular expression functions.
- [`github.com/ncruces/go-sqlite3/ext/statement`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/statement)
  creates [parameterized views](https://github.com/0x09/sqlite-statement-vtab).
- [`github.com/ncruces/go-sqlite3/ext/stats`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/stats)
  provides [statistics](https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html) functions.
- [`github.com/ncruces/go-sqlite3/ext/unicode`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/unicode)
  provides [Unicode aware](https://sqlite.org/src/dir/ext/icu) functions.
- [`github.com/ncruces/go-sqlite3/ext/uuid`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/uuid)
  generates [UUIDs](https://en.wikipedia.org/wiki/Universally_unique_identifier).
- [`github.com/ncruces/go-sqlite3/ext/zorder`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/zorder)
  maps multidimensional data to one dimension.