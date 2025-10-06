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

const DefaultPollInterval = 1 * time.Second

// VFS implements the SQLite VFS interface for Litestream.
// It is intended to be used for read replicas that read directly from S3.
type VFS struct {
	// PollInterval is the interval at which to poll the replica client for new
	// LTX files. The index will be fetched for the new files automatically.
	PollInterval time.Duration

	client litestream.ReplicaClient
	logger *slog.Logger
}

var _ vfs.VFS = &VFS{}

func NewVFS(client litestream.ReplicaClient, logger *slog.Logger) *VFS {
	return &VFS{
		client:       client,
		logger:       logger.With("vfs", "true"),
		PollInterval: DefaultPollInterval,
	}
}

func (fs *VFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// Temp journals, as used by the sorter, use SliceFile.
	if flags&vfs.OPEN_TEMP_JOURNAL != 0 {
		return &vfsutil.SliceFile{}, flags | vfs.OPEN_MEMORY, nil
	}
	// Refuse to open all other file types.
	if flags&vfs.OPEN_MAIN_DB == 0 {
		return nil, flags, sqlite3.CANTOPEN
	}

	f := liteFile{
		client:       fs.client,
		name:         name,
		pages:        map[uint32]ltx.PageIndexElem{},
		levels:       map[int]ltx.TXID{},
		logger:       fs.logger.With("name", name),
		pollInterval: fs.PollInterval,
	}

	// Build the page index so we can lookup individual pages.
	if err := f.buildIndex(context.Background()); err != nil {
		f.logger.Error("build index", "error", err)
		return nil, 0, err
	}
	return &f, flags | vfs.OPEN_READONLY, nil
}

func (vfs *VFS) Delete(name string, dirSync bool) error {
	// notest // used to delete journals
	return sqlite3.IOERR_DELETE_NOENT
}

func (vfs *VFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	// notest // used to check for journals
	return false, nil
}

func (vfs *VFS) FullPathname(name string) (string, error) {
	return name, nil
}

type liteFile struct {
	client litestream.ReplicaClient
	name   string

	pages  map[uint32]ltx.PageIndexElem
	levels map[int]ltx.TXID
	logger *slog.Logger
	conn   *sqlite3.Conn

	pageSize     uint32
	pageCount    uint32
	changeCount  uint32
	lastPoll     time.Time
	pollInterval time.Duration
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
		binary.BigEndian.PutUint32(data[24:28], f.changeCount)
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
		f.levels[info.Level] = info.MaxTXID
	}
	return nil
}

// pollReplicaClient fetches new LTX files from the replica client and updates
// the page index & the current position.
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
					return fmt.Errorf("non-contiguous ltx file: current=%s, next=%s-%s", nextTXID, info.MinTXID, info.MaxTXID)
				}

				// Read page index.
				idx, err := litestream.FetchPageIndex(ctx, f.client, info)
				if err != nil {
					return fmt.Errorf("fetch page index: %w", err)
				}

				// Update the page index & current position.
				for k, v := range idx {
					f.pageCount = max(f.pageCount, k)
					f.pages[k] = v
				}
				f.levels[level] = info.MaxTXID
				f.changeCount++
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
