# Go `"reader"` SQLite VFS

This package implements a `"reader"` SQLite VFS
that allows accessing any [`io.ReaderAt`](https://pkg.go.dev/io#ReaderAt)
as an immutable SQLite database.