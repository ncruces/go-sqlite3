package sqlite3

func deleteOnClose(f *os.File) {}

type vfsFileLocker = vfsNoopLocker
