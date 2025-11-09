# Litestream lightweight read-replicas

This package implements the **EXPERIMENTAL** `"litestream"` SQLite VFS
that offers Litestream [lightweight read-replicas](https://fly.io/blog/litestream-revamped/#lightweight-read-replicas).

See the [example](vfs_test.go) for how to use.

To improve performance,
increase `PollInterval` (and `MinLevel`) as much as you can,
and set [`PRAGMA cache_size=N`](https://www.sqlite.org/pragma.html#pragma_cache_size)
(or use `_pragma=cache_size(N)`).