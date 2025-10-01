# Go `mvcc` SQLite VFS

This package implements the **EXPERIMENTAL** `"mvcc"` in-memory SQLite VFS.

It has some benefits over the [`"memdb"`](../memdb/README.md) VFS:
- panics do not corrupt a shared database,
- single-writer not blocked by readers,
- readers never block,
- instant snapshots.

[`mvcc.TestDB`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/mvcc#TestDB)
is the preferred way to setup an in-memory database for testing
when you intend to leverage snapshots,
e.g. to setup many independent copies of a database,
such as one for each subtest.