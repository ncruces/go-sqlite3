package sqlite3

// Backup is a handle to an open BLOB.
//
// https://www.sqlite.org/c3ref/backup.html
type Backup struct {
	c      *Conn
	handle uint32
}

// BackupInit initializes a backup operation to copy the content of one database into another.
//
// BackupInit calls [Conn.Open] to open the SQLite database file dstURI,
// then initializes a backup that copies the content of srcDB to the "main" database in dstURI.
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupinit
func (c *Conn) BackupInit(srcDB, dstURI string) (*Backup, error) {
	return c.backupInit(srcDB, "main", 0)
}

func (c *Conn) backupInit(srcDB, dstDB string, handle uint32) (*Backup, error) {
	return nil, notImplErr
}

// Close finishes a backup operation.
//
// It is safe to close a nil, zero or closed Backup.
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupfinish
func (b *Backup) Close() error {
	if b == nil || b.handle == 0 {
		return nil
	}

	r := b.c.call(b.c.api.backupFinish, uint64(b.handle))

	b.handle = 0
	return b.c.error(r[0])
}

// Step copies up to nPage pages between the source and destination databases.
// If nPage is negative, all remaining source pages are copied.
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupstep
func (b *Backup) Step(nPage int) (done bool, err error) {
	r := b.c.call(b.c.api.backupStep, uint64(b.handle), uint64(nPage))
	if r[0] == _DONE {
		return true, nil
	}
	return false, b.c.error(r[0])
}

// Remaining returns the number of pages still to be backed up
// at the conclusion of the most recent [Backup.Step].
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backupremaining
func (b *Backup) Remaining() int {
	r := b.c.call(b.c.api.backupRemaining, uint64(b.handle))
	return int(r[0])
}

// PageCount returns the total number of pages in the source database
// at the conclusion of the most recent [Backup.Step].
//
// https://www.sqlite.org/c3ref/backup_finish.html#sqlite3backuppagecount
func (b *Backup) PageCount() int {
	r := b.c.call(b.c.api.backupFinish, uint64(b.handle))
	return int(r[0])
}
