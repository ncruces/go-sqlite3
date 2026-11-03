package tests

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/testcfg"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/go-sqlite3/vfs/adiantum"
	"github.com/ncruces/go-sqlite3/vfs/xts"
)

// TestEncryptedWALTornReads checks that readers never decrypt
// a WAL block torn by a concurrent read-update-write.
//
// WAL frames are not aligned to cipher blocks,
// so each commit rewrites the block holding the tail of
// the previous commit's last frame, while other connections
// may be reading that frame.
func TestEncryptedWALTornReads(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	ciphers := []struct {
		name string
		wrap func(vfs.VFS) vfs.VFS
	}{
		{"adiantum", func(base vfs.VFS) vfs.VFS { return adiantum.Wrap(base, nil) }},
		{"xts", func(base vfs.VFS) vfs.VFS { return xts.Wrap(base, nil) }},
	}

	for _, c := range ciphers {
		for _, unwrap := range []bool{true, false} {
			name := c.name + "-opaque"
			if unwrap {
				name = c.name + "-unwrap"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				vfs.Register("torn-"+name, c.wrap(tearVFS{vfs.Find(""), unwrap}))
				testTornReads(t, "torn-"+name)
			})
		}
	}
}

func testTornReads(t *testing.T, vfsName string) {
	const (
		commits = 100
		readers = 4
		window  = 64
	)

	ctx := testcfg.Context(t)
	uri := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "test.db")) +
		"?vfs=" + vfsName +
		"&hexkey=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	open := func() (*sqlite3.Conn, error) {
		db, err := sqlite3.OpenContext(ctx, uri)
		if err != nil {
			return nil, err
		}
		err = db.Exec(`PRAGMA busy_timeout=10000`)
		if err != nil {
			db.Close()
			return nil, err
		}
		return db, nil
	}

	value := func(id int64) string { return fmt.Sprintf("%0100d", id) }

	writer, err := open()
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	err = writer.Exec(`
		PRAGMA journal_mode=wal;
		PRAGMA synchronous=full;
		CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT NOT NULL);
	`)
	if err != nil {
		t.Fatal(err)
	}

	var (
		committed atomic.Int64
		reads     atomic.Int64
		failures  atomic.Int64
		firstErr  error
		errOnce   sync.Once
		wg        sync.WaitGroup
	)

	// read checks the last window rows committed.
	read := func(stmt *sqlite3.Stmt) (err error) {
		defer func() {
			// A torn page may decode as a huge record,
			// and SQLITE_NOMEM panics.
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()

		last := committed.Load()
		first := max(1, last-window+1)
		defer stmt.Reset()

		stmt.BindInt64(1, first)
		next := first
		for stmt.Step() {
			id := stmt.ColumnInt64(0)
			if id != next {
				return fmt.Errorf("got row %d, want %d", id, next)
			}
			if v := stmt.ColumnText(1); v != value(id) {
				return fmt.Errorf("row %d: wrong value %q", id, v)
			}
			next++
		}
		if err := stmt.Err(); err != nil {
			return err
		}
		if next <= last {
			return fmt.Errorf("missing row %d", next)
		}
		return nil
	}

	done := make(chan struct{})
	for range readers {
		wg.Go(func() {
			for {
				db, err := open()
				if err != nil {
					t.Error(err)
					return
				}
				stmt, _, err := db.Prepare(`SELECT id, v FROM t WHERE id >= ? ORDER BY id`)
				if err != nil {
					db.Close()
					t.Error(err)
					return
				}
				// A torn page stays in the page cache, so after a failure
				// reopen the connection to count distinct failures.
				for err == nil {
					select {
					case <-done:
						stmt.Close()
						db.Close()
						return
					default:
					}
					err = read(stmt)
					reads.Add(1)
				}
				failures.Add(1)
				errOnce.Do(func() { firstErr = err })
				stmt.Close()
				db.Close()
			}
		})
	}

	stmt, _, err := writer.Prepare(`INSERT INTO t (id, v) VALUES (?, ?)`)
	if err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= commits; id++ {
		stmt.BindInt64(1, id)
		stmt.BindText(2, value(id))
		if err := stmt.Exec(); err != nil {
			t.Fatal(err)
		}
		committed.Store(id)
	}
	stmt.Close()
	close(done)
	wg.Wait()

	if n := failures.Load(); n != 0 {
		t.Errorf("%d of %d reads failed, first error: %v", n, reads.Load(), firstErr)
	}
}

// tearVFS widens the window for torn reads in WAL files.
// Each write first stores a mix of old and new data,
// interleaved every 8 bytes, then the new data,
// like a slow write observed by a concurrent read would.
// Unless unwrap is set, the base file is hidden from wrappers.
type tearVFS struct {
	vfs.VFS
	unwrap bool
}

func (v tearVFS) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	file, flags, err := vfsutil.WrapOpenFilename(v.VFS, name, flags)
	if err == nil && flags&vfs.OPEN_WAL != 0 {
		if v.unwrap {
			file = &tearUnwrapFile{tearFile{file}}
		} else {
			file = &tearFile{file}
		}
	}
	return file, flags, err
}

type tearFile struct{ vfs.File }

func (f *tearFile) WriteAt(p []byte, off int64) (int, error) {
	torn := make([]byte, len(p))
	f.File.ReadAt(torn, off)
	for i := 0; i < len(p); i += 16 {
		copy(torn[i:min(i+8, len(p))], p[i:])
	}
	if _, err := f.File.WriteAt(torn, off); err != nil {
		return 0, err
	}
	time.Sleep(100 * time.Microsecond)
	return f.File.WriteAt(p, off)
}

type tearUnwrapFile struct{ tearFile }

func (f *tearUnwrapFile) Unwrap() vfs.File { return f.File }
