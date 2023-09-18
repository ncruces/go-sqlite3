# Go SQLite VFS API

This package implements the SQLite [OS Interface](https://www.sqlite.org/vfs.html) (aka VFS).

It replaces the default SQLite VFS with a pure Go implementation.

It also exposes interfaces that should allow you to implement your own custom VFSes.

## Portability

This package is tested on Linux, macOS and Windows,
but it should also work on FreeBSD and illumos
(code paths for those plaforms are tested on macOS and Linux, respectively).

In all platforms for which this package builds,
it should be safe to use it to access databases concurrently,
from multiple goroutines, processes, and
with _other_ implementations of SQLite.

If the package does not build for your platform,
you may try to use the `sqlite3_flock` and `sqlite3_nolock` build tags.
These are only minimally tested and concurrency test failures should be expected.

The `sqlite3_flock` tag uses
[BSD locks](https://man.freebsd.org/cgi/man.cgi?query=flock&sektion=2).
It should be safe to access databases concurrently from multiple goroutines and processes,
but **not** with _other_ implementations of SQLite
(_unless_ these are _also_ configured to use `flock`).

The `sqlite3_nolock` tag uses no locking at all.
Database corruption is the likely result from concurrent write access.