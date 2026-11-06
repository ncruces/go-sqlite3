package tests

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/ncruces/go-sqlite3/vfs/xts"
)

// On Linux, a buffered pread is not atomic with respect to a concurrent
// pwrite of the same page cache page (ext4, tmpfs, overlayfs).
// A reader can observe a mix of old and new bytes.
//
// For a plain database this is harmless:
// SQLite never reads bytes another connection is writing.
// For a block cipher it is not: a partial block write is a
// read-update-write of the whole block, so a reader of the
// unchanged part of the block gets a torn ciphertext,
// which decrypts to noise.
//
// The tornVFS makes the kernel race deterministic and wide:
// each write of a main database or WAL file first writes a mix
// of old and new bytes (interleaved every 8 bytes), sleeps,
// and only then writes the new data.
type tornVFS struct{ vfs.VFS }

func (t tornVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	return t.wrap(vfsutil.WrapOpen(t.VFS, name, flags))
}

func (t tornVFS) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	return t.wrap(vfsutil.WrapOpenFilename(t.VFS, name, flags))
}

func (tornVFS) wrap(file vfs.File, flags vfs.OpenFlag, err error) (vfs.File, vfs.OpenFlag, error) {
	if err != nil || flags&(vfs.OPEN_MAIN_DB|vfs.OPEN_WAL) == 0 {
		return file, flags, err
	}
	return &tornFile{File: file}, flags, nil
}

type tornFile struct{ vfs.File }

func (f *tornFile) WriteAt(p []byte, off int64) (int, error) {
	old := make([]byte, len(p))
	f.File.ReadAt(old, off) // missing bytes stay zero
	for i := range old {
		if i/8%2 != 0 {
			old[i] = p[i]
		}
	}
	if _, err := f.File.WriteAt(old, off); err != nil {
		return 0, err
	}
	time.Sleep(100 * time.Microsecond)
	return f.File.WriteAt(p, off)
}

func (f *tornFile) Unwrap() vfs.File { return f.File }

func (f *tornFile) SharedMemory() vfs.SharedMemory {
	return vfsutil.WrapSharedMemory(f.File)
}

var registerTorn = sync.OnceFunc(func() {
	vfs.Register("torn", tornVFS{vfs.Find("")})
	vfs.Register("torn-adiantum", adiantum.Wrap(vfs.Find("torn"), nil))
	vfs.Register("torn-xts", xts.Wrap(vfs.Find("torn"), nil))
})

func TestEncryptedWALTornReads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	registerTorn()

	for _, tt := range []struct {
		name       string
		vfs        string
		pageSize   int
		checkpoint bool
	}{
		// One writer, concurrent readers.
		{"adiantum", "torn-adiantum", 4096, false},
		{"xts", "torn-xts", 4096, false},
		// Add a connection that only checkpoints:
		// a torn read by the checkpointer corrupts the database file.
		{"adiantum/checkpoint", "torn-adiantum", 4096, true},
		{"xts/checkpoint", "torn-xts", 4096, true},
		// Pages smaller than the cipher block share blocks
		// in the database file; checkpoints rewrite them.
		{"adiantum/512", "torn-adiantum", 512, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testTornReads(t, tt.vfs, tt.pageSize, tt.checkpoint)
		})
	}
}

const (
	tornCommits = 200
	tornReaders = 4
	tornWindow  = 64
)

func tornURI(t testing.TB, vfs string) string {
	return "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "torn.db")) +
		"?vfs=" + vfs + "&hexkey=" + strings.Repeat("ab", 32)
}

func tornOpen(t testing.TB, uri string) *sqlite3.Conn {
	t.Helper()
	db, err := sqlite3.Open(uri)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.BusyTimeout(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	return db
}

func testTornReads(t *testing.T, vfsName string, pageSize int, checkpoint bool) {
	uri := tornURI(t, vfsName)

	w := tornOpen(t, uri)
	defer w.Close()
	err := w.Exec(fmt.Sprintf(`
		PRAGMA page_size=%d;
		PRAGMA journal_mode=wal;
		PRAGMA synchronous=full;
		CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)`, pageSize))
	if err != nil {
		t.Fatal(err)
	}

	var last, reads, fails atomic.Int64
	var done atomic.Bool
	var wg sync.WaitGroup
	for range tornReaders {
		wg.Go(func() {
			tornReader(t, uri, &last, &reads, &fails, &done)
		})
	}
	if checkpoint {
		wg.Go(func() {
			db := tornOpen(t, uri)
			defer db.Close()
			for !done.Load() {
				err := db.Exec(`PRAGMA wal_checkpoint(PASSIVE)`)
				if err != nil && !errors.Is(err, sqlite3.BUSY) {
					t.Error(err)
					return
				}
			}
		})
	}

	ins, _, err := w.Prepare(`INSERT INTO t VALUES (?1, printf('%0100d', ?1))`)
	if err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= tornCommits; id++ {
		ins.BindInt64(1, id)
		if err := ins.Exec(); err != nil {
			t.Fatal(err)
		}
		last.Store(id)
	}
	ins.Close()
	done.Store(true)
	wg.Wait()

	t.Logf("%d of %d reads failed", fails.Load(), reads.Load())
	if fails.Load() != 0 {
		t.Fail()
	}

	// Check for persistent corruption with a fresh connection.
	w.Close()
	db := tornOpen(t, uri)
	defer db.Close()
	st, _, err := db.Prepare(`PRAGMA integrity_check`)
	if err != nil {
		t.Fatal(err)
	}
	if st.Step() && st.ColumnText(0) != "ok" {
		t.Error(st.ColumnText(0))
	}
	st.Close()
	st, _, err = db.Prepare(`SELECT count(*) FROM t WHERE v = printf('%0100d', id)`)
	if err != nil {
		t.Fatal(err)
	}
	if st.Step() && st.ColumnInt64(0) != tornCommits {
		t.Errorf("got %d rows, want %d", st.ColumnInt64(0), tornCommits)
	}
	st.Close()
}

// tornReader checks that the last tornWindow rows are all present and intact.
// After a failure it reconnects, because corrupt pages stay in the page cache.
func tornReader(t *testing.T, uri string, last, reads, fails *atomic.Int64, done *atomic.Bool) {
	for !done.Load() {
		db := tornOpen(t, uri)
		st, _, err := db.Prepare(`SELECT count(*) FROM t
			WHERE id > ?1 - ` + fmt.Sprint(tornWindow) + ` AND id <= ?1 AND v = printf('%0100d', id)`)
		if err != nil {
			t.Error(err)
			db.Close()
			return
		}
		for !done.Load() {
			n := last.Load()
			st.BindInt64(1, n)
			reads.Add(1)
			if !st.Step() || st.ColumnInt64(0) != min(n, tornWindow) {
				err := st.Err()
				if err == nil {
					err = fmt.Errorf("wrong result: %d of %d rows", st.ColumnInt64(0), min(n, tornWindow))
				}
				if fails.Add(1) <= 3 {
					t.Log(err)
				}
				break
			}
			st.Reset()
		}
		st.Close()
		db.Close()
	}
}
