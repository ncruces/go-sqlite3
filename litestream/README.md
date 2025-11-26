# Litestream in-process replication and lightweight read-replicas

This package adds **EXPERIMENTAL** support for in-process [Litestream](https://litestream.io/).

## Lightweight read-replicas

The `"litestream"` SQLite VFS implements Litestream
[lightweight read-replicas](https://fly.io/blog/litestream-revamped/#lightweight-read-replicas).

See the [example](example_test.go) for how to use.

To improve performance,
increase `PollInterval` (and `MinLevel`) as much as you can,
and set [`PRAGMA cache_size=N`](https://www.sqlite.org/pragma.html#pragma_cache_size)
(or use `_pragma=cache_size(N)`).

## In-process replication

For disaster recovery, it is probably best if you run Litestream as a separate background process,
as recommended by the [tutorial](https://litestream.io/getting-started/).

However, running Litestream as a background process requires
compatible locking and cross-process shared memory WAL
(see our [support matrix](https://github.com/ncruces/go-sqlite3/wiki/Support-matrix)).

If your OS lacks locking or shared memory support,
you can use `NewPrimary` with the `sqlite3_dotlk` build tag to setup in-process replication.
