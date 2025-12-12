# Litestream lightweight read-replicas

This package implements the **EXPERIMENTAL** `"litestream"` SQLite VFS
that offers Litestream [lightweight read-replicas](https://fly.io/blog/litestream-revamped/#lightweight-read-replicas).

See the [example](example_test.go) for how to use.

Our `PRAGMA litestream_time` accepts:
- Go [duration strings](https://pkg.go.dev/time#ParseDuration)
- SQLite [time values](https://sqlite.org/lang_datefunc.html#time_values)
- SQLite [time modifiers 1 through 13](https://sqlite.org/lang_datefunc.html#modifiers)
