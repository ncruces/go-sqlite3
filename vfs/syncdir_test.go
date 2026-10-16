package vfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// syncDirFile opens a new WAL file in its own temporary directory as a
// vfsFile that still owes its directory a sync, and swaps syncDir for a
// recorder that appends the name of each directory it is handed and answers
// with ret. The recorder is restored when the test ends.
func syncDirFile(t *testing.T, ret error) (f *vfsFile, dir string, got *[]string) {
	t.Helper()
	dir = t.TempDir()
	file, err := os.OpenFile(filepath.Join(dir, "test.db-wal"), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })

	got = new([]string)
	old := syncDir
	syncDir = func(d *os.File) error {
		fi, err := d.Stat()
		if err != nil || !fi.IsDir() {
			t.Errorf("syncDir was handed %q, which is not an open directory (%v)", d.Name(), err)
		}
		*got = append(*got, d.Name())
		return ret
	}
	t.Cleanup(func() { syncDir = old })

	f = &vfsFile{File: file, flags: OPEN_WAL | OPEN_CREATE | _FLAG_SYNC_DIR}
	return f, dir, got
}

// TestSyncFlushesTheDirectory holds Sync to SQLite's unixSync: the first Sync
// of a new journal or WAL also syncs the directory that holds its entry, once,
// and neither a directory that will not open nor a failed directory sync
// fails the Sync.
func TestSyncFlushesTheDirectory(t *testing.T) {
	if !isUnix {
		t.Skip("the directory sync runs only on unix")
	}

	t.Run("dir", func(t *testing.T) {
		f, dir, got := syncDirFile(t, nil)
		if err := f.Sync(SYNC_NORMAL); err != nil {
			t.Fatal(err)
		}
		if len(*got) != 1 || (*got)[0] != dir {
			t.Fatalf("Sync synced the directories %q, want exactly [%q]", *got, dir)
		}
	})

	t.Run("once", func(t *testing.T) {
		f, dir, got := syncDirFile(t, nil)
		if err := f.Sync(SYNC_NORMAL); err != nil {
			t.Fatal(err)
		}
		if f.flags&_FLAG_SYNC_DIR != 0 {
			t.Error("the first Sync left the directory-sync flag set")
		}
		if err := f.Sync(SYNC_NORMAL); err != nil {
			t.Fatal(err)
		}
		if len(*got) != 1 || (*got)[0] != dir {
			t.Fatalf("two Syncs synced the directories %q, want exactly [%q] once", *got, dir)
		}
	})

	t.Run("unopenable-dir", func(t *testing.T) {
		f, dir, got := syncDirFile(t, nil)
		if err := os.Chmod(dir, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0o700) })
		if d, err := os.Open(dir); err == nil {
			d.Close()
			t.Skip("a directory with mode 000 still opens here (a superuser?)")
		}
		if err := f.Sync(SYNC_NORMAL); err != nil {
			t.Fatalf("Sync failed on a directory that will not open: %v", err)
		}
		if len(*got) != 0 {
			t.Errorf("Sync synced the directories %q, which did not open", *got)
		}
		if f.flags&_FLAG_SYNC_DIR != 0 {
			t.Error("Sync left the directory-sync flag set")
		}
	})

	t.Run("error-ignored", func(t *testing.T) {
		f, dir, got := syncDirFile(t, errors.New("planted directory sync failure"))
		if err := f.Sync(SYNC_NORMAL); err != nil {
			t.Fatalf("a failed directory sync failed Sync: %v", err)
		}
		if len(*got) != 1 || (*got)[0] != dir {
			t.Fatalf("Sync synced the directories %q, want exactly [%q]", *got, dir)
		}
		if f.flags&_FLAG_SYNC_DIR != 0 {
			t.Error("Sync left the directory-sync flag set")
		}
	})
}
