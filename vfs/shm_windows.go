//go:build !sqlite3_dotlk

package vfs

import (
	"io"
	"os"
	"sync/atomic"

	"golang.org/x/sys/windows"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

// On Windows 10 1803+ / Server 2019+ the WAL-index is mapped directly into
// the wasm linear memory (like the unix build maps it with MAP_FIXED):
// SQLite then works on genuinely shared, always-current memory, which
// wal.c's protocol assumes.
//
// wal.c reads and writes the index BETWEEN xShmLock calls and relies on
// seeing other connections' updates live — e.g. the checkpointer probes each
// aReadMark slot and, when the probe loses to a live reader, trusts the mark
// value it read moments earlier. Any scheme that synchronizes copies of the
// index only at lock boundaries serves stale views on exactly such paths;
// that reliably corrupted databases under concurrent write load (torn reads
// of database pages being backfilled past a reader's snapshot, checkpoints
// driven by hole-ridden hash tables, ...).
//
// Below Windows 10 1803 the placeholder APIs are unavailable, so the index
// cannot be mapped into linear memory. Rather than fall back to a
// copy-on-lock-boundary scheme that cannot make wal.c's between-lock reads
// coherent, shared-memory WAL is refused there: such a build must use
// [WAL without shared-memory] via [EXCLUSIVE locking mode], the same as
// platforms without shared-memory support.
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode

const (
	_WALINDEX_PGSZ = 32768
	_SHM_VIEW      = 2 * _WALINDEX_PGSZ // regions are mapped in 64K pairs
)

type shmView struct {
	block ptr_t // sqlite3_malloc'd block the view was carved from
	addr  ptr_t // 64K-aligned start of the mapped 64K view
}

type vfsShm struct {
	*os.File
	wrp      *sqlite3_wrap.Wrapper
	path     string
	views    []shmView // one per mapped region pair
	fileLock bool
}

func (s *vfsShm) Close() error {
	// Unmap views.
	for _, v := range s.views {
		if err := s.wrp.UnmapFileRegion(uintptr(v.addr), _SHM_VIEW); err != nil {
			// The view is still mapped over the block: freeing it would
			// let the allocator hand out file-backed memory, and writes
			// through it would corrupt the wal-index for every other
			// connection. Leak the block instead.
			continue
		}
		s.wrp.Xsqlite3_free(int32(v.block))
	}
	s.views = nil

	// Close the file.
	return s.File.Close()
}

func (s *vfsShm) shmOpen() error {
	if s.fileLock {
		return nil
	}
	if s.File == nil {
		f, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return sysError{err, _CANTOPEN}
		}
		s.fileLock = false
		s.File = f
	}

	// Dead man's switch.
	if osWriteLock(s.File, _SHM_DMS, 1, 0) == nil {
		err := s.Truncate(0)
		osUnlock(s.File, _SHM_DMS, 1)
		if err != nil {
			return sysError{err, _IOERR_SHMOPEN}
		}
	}
	err := osReadLock(s.File, _SHM_DMS, 1, 0)
	s.fileLock = err == nil
	return err
}

func (s *vfsShm) shmMap(wrp *sqlite3_wrap.Wrapper, id, size int32, extend bool) (_ ptr_t, err error) {
	// Ensure pages are reasonably sized.
	if size != _WALINDEX_PGSZ || (windows.Getpagesize() > int(size)*2) {
		return 0, _IOERR_SHMMAP
	}
	if s.wrp == nil {
		s.wrp = wrp
	}
	// Below Windows 10 1803 the index cannot be mapped into linear memory;
	// refuse shared-memory WAL (WAL then needs EXCLUSIVE locking mode).
	if !wrp.CanMapFiles() {
		return 0, _IOERR_SHMMAP
	}
	if err := s.shmOpen(); err != nil {
		return 0, err
	}
	return s.shmMapDirect(id, size, extend)
}

// shmMapDirect maps the region into the wasm linear memory, in 64K pairs
// (both the map address and the file offset must be allocation-granularity
// aligned). The view is carved out of a sqlite3_malloc'd block: the
// allocator keeps the address range reserved while the underlying pages are
// replaced by the file view.
func (s *vfsShm) shmMapDirect(id, size int32, extend bool) (ptr_t, error) {
	pair := int(id) / 2

	// Check if the file covers the requested region.
	o, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, sysError{err, _IOERR_SHMSIZE}
	}
	if n := (int64(id) + 1) * int64(size); n > o {
		if !extend {
			return 0, nil
		}
	}

	for len(s.views) <= pair {
		p := len(s.views)
		// The pair's view spans [p*64K, (p+1)*64K) of the file.
		// This can grow the file to the pair boundary even when
		// extend is false (e.g. a 32K file, region 0 requested),
		// which deviates from the xShmMap contract. It is benign:
		// the extension is zero-filled, zero hash pages read as
		// empty, and no reader consults index pages beyond the
		// mxFrame published in the header — the same state a
		// legitimate writer extension leaves before publishing.
		if n := int64(p+1) * _SHM_VIEW; n > o {
			if err := osAllocate(s.File, n); err != nil {
				return 0, sysError{err, _IOERR_SHMSIZE}
			}
		}
		// Carve a 64K-aligned 64K hole from a malloc'd block
		// (2×64K guarantees an aligned 64K fits at any block offset).
		block := s.wrp.Xsqlite3_malloc64(4 * _WALINDEX_PGSZ)
		if block == 0 {
			panic(errutil.OOMErr)
		}
		// Convert through uint32: the wasm pointer is returned as int32,
		// and above 2 GiB of linear memory bit 31 is set — a direct
		// uintptr conversion would sign-extend and wreck the arithmetic.
		aligned := (uintptr(uint32(block)) + _SHM_VIEW - 1) &^ (_SHM_VIEW - 1)
		if err := s.wrp.MapFileRegion(s.File, int64(p)*_SHM_VIEW, aligned, _SHM_VIEW); err != nil {
			s.wrp.Xsqlite3_free(int32(block))
			return 0, sysError{err, _IOERR_SHMMAP}
		}
		s.views = append(s.views, shmView{block: ptr_t(block), addr: ptr_t(aligned)})
	}

	return s.views[pair].addr + ptr_t(int(id)%2)*_WALINDEX_PGSZ, nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) (err error) {
	if s.File == nil {
		return _IOERR_SHMLOCK
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		return osUnlock(s.File, _SHM_BASE+uint32(offset), uint32(n))
	case flags&_SHM_SHARED != 0:
		return osReadLock(s.File, _SHM_BASE+uint32(offset), uint32(n), 0)
	case flags&_SHM_EXCLUSIVE != 0:
		return osWriteLock(s.File, _SHM_BASE+uint32(offset), uint32(n), 0)
	default:
		panic(errutil.AssertErr())
	}
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.File == nil {
		return
	}

	// Close the file (also unmaps views).
	s.Close()
	s.File = nil
	s.fileLock = false
	if delete {
		os.Remove(s.path)
	}
}

func (s *vfsShm) shmBarrier() {
	// The index is genuinely shared: a memory fence suffices, as on unix.
	var b atomic.Bool
	b.Swap(true)
}
