# Go `mvcc` SQLite VFS

This package implements the **EXPERIMENTAL** `"mvcc"` in-memory SQLite VFS.

It has some benefits over the [`"memdb"`](../memdb/README.md) VFS:
- panics do not corrupt a shared database;
- single-writer not blocked by readers,
- readers never block,
- instant snapshots.