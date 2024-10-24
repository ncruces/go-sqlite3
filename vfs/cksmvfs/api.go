// Package cksmvfs wraps an SQLite VFS to help detect database corruption.
//
// The "cksmvfs" [vfs.VFS] wraps the default VFS adding an 8-byte checksum
// to the end of every page in an SQLite database.
// The checksum is added as each page is written
// and verified as each page is read.
// The checksum is intended to help detect database corruption
// caused by random bit-flips in the mass storage device.
//
// This implementation is fully compatible with SQLite's
// [Checksum VFS Shim].
//
// [Checksum VFS Shim]: https://sqlite.org/cksumvfs.html
package cksmvfs

import (
	"fmt"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/vfs"
)

func init() {
	vfs.Register("cksmvfs", Wrap(vfs.Find("")))
}

// Wrap wraps a base VFS to create a checksumming VFS.
func Wrap(base vfs.VFS) vfs.VFS {
	return &cksmVFS{VFS: base}
}

// Enable enables checksums on a database.
func Enable(db *sqlite3.Conn, schema string) error {
	// Set reserve bytes to 8.
	r, err := db.FileControl(schema, sqlite3.FCNTL_RESERVE_BYTES)
	if err != nil {
		return err
	}
	if r == 8 {
		// Correct value, enabled.
		return nil
	}
	if r == 0 {
		// Default value, enable.
		r, err = db.FileControl(schema, sqlite3.FCNTL_RESERVE_BYTES, 8)
	}
	if err != nil {
		return err
	}
	if r != 8 {
		// Invalid value.
		return fmt.Errorf("cksmvfs: reserve bytes must be 8, is: %d", r)
	}

	// VACUUM the database.
	if schema != "" {
		err = db.Exec("VACUUM " + sqlite3.QuoteIdentifier(schema))
	} else {
		err = db.Exec("VACUUM")
	}
	if err != nil {
		return err
	}

	// Checkpoint the WAL.
	_, _, err = db.WALCheckpoint(schema, sqlite3.CHECKPOINT_RESTART)
	return err
}
