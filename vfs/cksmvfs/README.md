# Go `cksmvfs` SQLite VFS

This package wraps an SQLite VFS to help detect database corruption.

The `"cksmvfs"` VFS wraps the default SQLite VFS adding an 8-byte checksum
to the end of every page in an SQLite database.\
The checksum is added as each page is written
and verified as each page is read.\
The checksum is intended to help detect database corruption
caused by random bit-flips in the mass storage device.

This implementation is compatible with SQLite's
[Checksum VFS Shim](https://sqlite.org/cksumvfs.html).

> [!IMPORTANT]
> [Checksums](https://en.wikipedia.org/wiki/Checksum)
> are meant to protect against _silent data corruption_ (bit rot).
> They do not offer _authenticity_ (i.e. protect against _forgery_),
> nor prevent _silent loss of durability_.
> Checkpoint WAL mode databases to improve durabiliy.