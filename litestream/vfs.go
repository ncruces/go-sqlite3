package litestream

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/benbjohnson/litestream"
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/superfly/ltx"
)

type liteVFS struct{}

func (liteVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// notest // OpenFilename is called instead
	if name == "" {
		return liteVFS{}.OpenFilename(nil, flags)
	}
	return nil, flags, sqlite3.CANTOPEN
}

func (liteVFS) OpenFilename(name *vfs.Filename, flags vfs.OpenFlag) (file vfs.File, _ vfs.OpenFlag, err error) {
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
	if db, ok := liteDBs[name.String()]; ok {
		f := liteFile{
			client:       db.client,
			pages:        map[uint32]ltx.PageIndexElem{},
			levels:       map[int]ltx.TXID{},
			logger:       db.logger,
			pollInterval: DefaultPollInterval,
		}

		if poll := name.URIParameter("_poll_interval"); poll != "" {
			f.pollInterval, err = time.ParseDuration(poll)
			if err != nil {
				f.logger.Error("parse _poll_interval", "error", err)
				return nil, 0, err
			}
		}

		// Build the page index so we can lookup individual pages.
		if err := f.buildIndex(context.Background()); err != nil {
			f.logger.Error("build index", "error", err)
			return nil, 0, err
		}
		return &f, flags | vfs.OPEN_READONLY, nil
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
	client litestream.ReplicaClient

	pages  map[uint32]ltx.PageIndexElem
	levels map[int]ltx.TXID
	logger *slog.Logger
	conn   *sqlite3.Conn

	lastPoll     time.Time
	pollInterval time.Duration

	maxTXID   ltx.TXID
	pageSize  uint32
	pageCount uint32
}

func (f *liteFile) Close() error { return nil }

func (f *liteFile) ReadAt(p []byte, off int64) (n int, err error) {
	pgno := uint32(1)
	if off >= 512 {
		pgno += uint32(off / int64(f.pageSize))
	}

	elem, ok := f.pages[pgno]
	if !ok {
		return 0, io.EOF
	}

	ctx := context.Background()
	if f.conn != nil {
		ctx = f.conn.GetInterrupt()
	}

	_, data, err := litestream.FetchPage(ctx, f.client, elem.Level, elem.MinTXID, elem.MaxTXID, elem.Offset, elem.Size)
	if err != nil {
		f.logger.Error("fetch page", "error", err)
		return 0, err
	}

	// Update the first page to pretend we are in journal mode,
	// load the page size and track changes to the database.
	if pgno == 1 && len(data) >= 100 && (false ||
		data[18] == 2 && data[19] == 2 ||
		data[18] == 3 && data[19] == 3) {
		data[18], data[19] = 0x01, 0x01
		binary.BigEndian.PutUint32(data[24:28], uint32(f.maxTXID))
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
	size = int64(f.pageCount) * int64(f.pageSize)
	return
}

func (f *liteFile) Lock(lock vfs.LockLevel) error {
	if lock >= vfs.LOCK_RESERVED {
		return sqlite3.IOERR_LOCK
	}
	return f.pollReplicaClient()
}

func (f *liteFile) Unlock(lock vfs.LockLevel) error {
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

func (f *liteFile) SetDB(conn any) {
	f.conn = conn.(*sqlite3.Conn)
}

func (f *liteFile) buildIndex(ctx context.Context) error {
	infos, err := litestream.CalcRestorePlan(ctx, f.client, 0, time.Time{}, f.logger)
	if err != nil {
		if !errors.Is(err, litestream.ErrTxNotAvailable) {
			return fmt.Errorf("calc restore plan: %w", err)
		}
		return nil
	}

	for _, info := range infos {
		err := f.updateIndex(ctx, info)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *liteFile) pollReplicaClient() error {
	// Limit polling interval.
	if time.Since(f.lastPoll) < f.pollInterval {
		return nil
	}
	f.lastPoll = time.Now()

	ctx := context.Background()
	if f.conn != nil {
		ctx = f.conn.GetInterrupt()
	}

	for level := range litestream.SnapshotLevel + 1 {
		err := func() error {
			nextTXID := f.levels[level] + 1

			// Start reading from the next LTX file after the current position.
			itr, err := f.client.LTXFiles(ctx, level, nextTXID)
			if err != nil {
				return fmt.Errorf("ltx files: %w", err)
			}
			defer itr.Close()

			// Build an update across all new LTX files.
			for itr.Next() {
				info := itr.Item()

				// Ensure we are fetching the next transaction from our current position.
				if nextTXID != 1 && nextTXID != info.MinTXID {
					return fmt.Errorf("non-contiguous ltx file: current=%s, next=%s-%s",
						nextTXID, info.MinTXID, info.MaxTXID)
				}

				err := f.updateIndex(ctx, info)
				if err != nil {
					return err
				}
			}

			return itr.Close()
		}()
		if err != nil {
			f.logger.Error("cannot poll replica", "error", err)
			return err
		}
	}
	return nil
}

func (f *liteFile) updateIndex(ctx context.Context, info *ltx.FileInfo) error {
	// Read page index.
	idx, err := litestream.FetchPageIndex(ctx, f.client, info)
	if err != nil {
		return fmt.Errorf("fetch page index: %w", err)
	}

	// Replace pages in overall index with new pages.
	for k, v := range idx {
		f.pageCount = max(f.pageCount, k)
		f.pages[k] = v
	}
	f.maxTXID = max(f.maxTXID, info.MaxTXID)
	f.levels[info.Level] = info.MaxTXID
	return nil
}
