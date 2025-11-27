package litestream

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/benbjohnson/litestream"
	"github.com/superfly/ltx"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/wbt"
)

type liteVFS struct{}

func (liteVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// Temp journals, as used by the sorter, use SliceFile.
	if flags&vfs.OPEN_TEMP_JOURNAL != 0 {
		return &vfsutil.SliceFile{}, flags | vfs.OPEN_MEMORY, nil
	}
	// Refuse to open all other file types.
	if flags&vfs.OPEN_MAIN_DB == 0 {
		return nil, flags, sqlite3.CANTOPEN
	}

	liteMtx.RLock()
	defer liteMtx.RUnlock()
	if db, ok := liteDBs[name]; ok {
		// Build the page index so we can lookup individual pages.
		if err := db.buildIndex(context.Background()); err != nil {
			db.opts.Logger.Error("build index", "error", err)
			return nil, 0, err
		}
		return &liteFile{db: db}, flags | vfs.OPEN_READONLY, nil
	}
	return nil, flags, sqlite3.CANTOPEN
}

func (liteVFS) Delete(name string, dirSync bool) error {
	// notest // used to delete journals
	return sqlite3.IOERR_DELETE_NOENT
}

func (liteVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	// notest // used to check for journals
	return false, nil
}

func (liteVFS) FullPathname(name string) (string, error) {
	return name, nil
}

type liteFile struct {
	db       *liteDB
	conn     *sqlite3.Conn
	pages    *pageIndex
	txid     ltx.TXID
	pageSize uint32
}

func (f *liteFile) Close() error { return nil }

func (f *liteFile) ReadAt(p []byte, off int64) (n int, err error) {
	ctx := f.context()
	pages, txid := f.pages, f.txid
	if pages == nil {
		pages, txid, err = f.db.pollReplica(ctx)
	}
	if err != nil {
		return 0, err
	}

	pgno := uint32(1)
	if off >= 512 {
		pgno += uint32(off / int64(f.pageSize))
	}

	elem, ok := pages.Get(pgno)
	if !ok {
		return 0, io.EOF
	}

	data, err := f.db.cache.getOrFetch(pgno, elem.MaxTXID, func() (any, error) {
		_, data, err := litestream.FetchPage(ctx, f.db.client, elem.Level, elem.MinTXID, elem.MaxTXID, elem.Offset, elem.Size)
		return data, err
	})
	if err != nil {
		f.db.opts.Logger.Error("fetch page", "error", err)
		return 0, err
	}

	// Update the first page to pretend we are in journal mode,
	// load the page size and track changes to the database.
	if pgno == 1 && len(data) >= 100 &&
		data[18] >= 1 && data[19] >= 1 &&
		data[18] <= 3 && data[19] <= 3 {
		data[18], data[19] = 0x01, 0x01
		binary.BigEndian.PutUint32(data[24:28], uint32(txid))
		f.pageSize = uint32(256 * binary.LittleEndian.Uint16(data[16:18]))
	}

	n = copy(p, data[off%int64(len(data)):])
	return n, nil
}

func (f *liteFile) WriteAt(b []byte, off int64) (n int, err error) {
	// notest // OPEN_READONLY
	return 0, sqlite3.IOERR_WRITE
}

func (f *liteFile) Truncate(size int64) error {
	// notest // OPEN_READONLY
	return sqlite3.IOERR_TRUNCATE
}

func (f *liteFile) Sync(flag vfs.SyncFlag) error {
	// notest // OPEN_READONLY
	return sqlite3.IOERR_FSYNC
}

func (f *liteFile) Size() (size int64, err error) {
	if max := f.pages.Max(); max != nil {
		size = int64(max.Key()) * int64(f.pageSize)
	}
	return
}

func (f *liteFile) Lock(lock vfs.LockLevel) (err error) {
	if lock >= vfs.LOCK_RESERVED {
		return sqlite3.IOERR_LOCK
	}
	f.pages, f.txid, err = f.db.pollReplica(f.context())
	return err
}

func (f *liteFile) Unlock(lock vfs.LockLevel) error {
	f.pages, f.txid = nil, 0
	return nil
}

func (f *liteFile) CheckReservedLock() (bool, error) {
	// notest // used to check for hot journals
	return false, nil
}

func (f *liteFile) SectorSize() int {
	// notest // safe default
	return 0
}

func (f *liteFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	// notest // safe default
	return 0
}

func (f *liteFile) Pragma(name, value string) (string, error) {
	switch name {
	case "litestream_txid":
		txid := f.txid
		if txid == 0 {
			// Outside transaction.
			f.db.mtx.Lock()
			txid = f.db.txids[0]
			f.db.mtx.Unlock()
		}
		return txid.String(), nil

	case "litestream_lag":
		f.db.mtx.Lock()
		lastPoll := f.db.lastPoll
		f.db.mtx.Unlock()

		if lastPoll.IsZero() {
			// Never polled successfully.
			return "-1", nil
		}
		lag := time.Since(lastPoll) / time.Second
		return strconv.FormatInt(int64(lag), 10), nil
	}

	return "", sqlite3.NOTFOUND
}

func (f *liteFile) SetDB(conn any) {
	f.conn = conn.(*sqlite3.Conn)
}

func (f *liteFile) context() context.Context {
	if f.conn != nil {
		return f.conn.GetInterrupt()
	}
	return context.Background()
}

type liteDB struct {
	client   litestream.ReplicaClient
	opts     ReplicaOptions
	cache    pageCache
	pages    *pageIndex // +checklocks:mtx
	lastPoll time.Time  // +checklocks:mtx
	txids    levelTXIDs // +checklocks:mtx
	mtx      sync.Mutex
}

func (f *liteDB) buildIndex(ctx context.Context) error {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	// Skip if we already have an index.
	if f.pages != nil {
		return nil
	}

	// Build the index from scratch from a Litestream restore plan.
	infos, err := litestream.CalcRestorePlan(ctx, f.client, 0, time.Time{}, f.opts.Logger)
	if err != nil {
		if !errors.Is(err, litestream.ErrTxNotAvailable) {
			return fmt.Errorf("calc restore plan: %w", err)
		}
		return nil
	}

	for _, info := range infos {
		err := f.updateInfo(ctx, info)
		if err != nil {
			return err
		}
	}

	f.lastPoll = time.Now()
	return nil
}

func (f *liteDB) pollReplica(ctx context.Context) (*pageIndex, ltx.TXID, error) {
	f.mtx.Lock()
	defer f.mtx.Unlock()

	// Limit polling interval.
	if time.Since(f.lastPoll) < f.opts.PollInterval {
		return f.pages, f.txids[0], nil
	}

	for level := range []int{0, 1, litestream.SnapshotLevel} {
		if err := f.updateLevel(ctx, level); err != nil {
			f.opts.Logger.Error("cannot poll replica", "error", err)
			return nil, 0, err
		}
	}

	f.lastPoll = time.Now()
	return f.pages, f.txids[0], nil
}

// +checklocks:f.mtx
func (f *liteDB) updateLevel(ctx context.Context, level int) error {
	var nextTXID ltx.TXID
	// Snapshots must start from scratch,
	// other levels can start from where they were left.
	if level != litestream.SnapshotLevel {
		nextTXID = f.txids[level] + 1
	}

	// Start reading from the next LTX file after the current position.
	itr, err := f.client.LTXFiles(ctx, level, nextTXID, false)
	if err != nil {
		return fmt.Errorf("ltx files: %w", err)
	}
	defer itr.Close()

	// Build an update across all new LTX files.
	for itr.Next() {
		info := itr.Item()

		// Skip LTX files already fully loaded into the index.
		if info.MaxTXID <= f.txids[level] {
			continue
		}

		err := f.updateInfo(ctx, info)
		if err != nil {
			return err
		}
	}
	if err := itr.Err(); err != nil {
		return err
	}
	return itr.Close()
}

// +checklocks:f.mtx
func (f *liteDB) updateInfo(ctx context.Context, info *ltx.FileInfo) error {
	idx, err := litestream.FetchPageIndex(ctx, f.client, info)
	if err != nil {
		return fmt.Errorf("fetch page index: %w", err)
	}

	// Replace pages in the index with new pages.
	for k, v := range idx {
		// Patch avoids mutating the index for an unmodified page.
		f.pages = f.pages.Patch(k, func(node *pageIndex) (ltx.PageIndexElem, bool) {
			return v, node == nil || v != node.Value()
		})
	}

	// Track the MaxTXID for each level.
	maxTXID := &f.txids[info.Level]
	*maxTXID = max(*maxTXID, info.MaxTXID)
	f.txids[0] = max(f.txids[0], *maxTXID)
	return nil
}

// Type aliases; these are a mouthful.
type pageIndex = wbt.Tree[uint32, ltx.PageIndexElem]
type levelTXIDs = [litestream.SnapshotLevel + 1]ltx.TXID
