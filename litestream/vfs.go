package litestream

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/benbjohnson/litestream"
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/superfly/ltx"
)

const (
	DefaultPollInterval = 1 * time.Second
)

// VFS implements the SQLite VFS interface for Litestream.
// It is intended to be used for read replicas that read directly from S3.
type VFS struct {
	client litestream.ReplicaClient
	logger *slog.Logger

	// PollInterval is the interval at which to poll the replica client for new
	// LTX files. The index will be fetched for the new files automatically.
	PollInterval time.Duration
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
	fs.logger.Info("opening file", "name", name, "flags", flags)

	// Temp journals, as used by the sorter, use SliceFile.
	if flags&vfs.OPEN_TEMP_JOURNAL != 0 {
		return &vfsutil.SliceFile{}, flags | vfs.OPEN_MEMORY, nil
	}
	// Refuse to open all other file types.
	if flags&vfs.OPEN_MAIN_DB == 0 {
		return nil, flags, sqlite3.CANTOPEN
	}

	// TODO: Clone client w/ new path based on name.

	f := liteFile{
		client:       fs.client,
		name:         name,
		index:        make(map[uint32]ltx.PageIndexElem),
		logger:       fs.logger.With("name", name),
		pollInterval: fs.PollInterval,
	}

	// Build the page index so we can lookup individual pages.
	if err := f.buildIndex(context.Background()); err != nil {
		f.logger.Error("cannot build index", "error", err)
		return nil, 0, fmt.Errorf("cannot build index: %w", err)
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

	index  map[uint32]ltx.PageIndexElem
	logger *slog.Logger
	conn   *sqlite3.Conn

	pos          ltx.Pos
	pageSize     uint32
	pageCount    uint32
	changeCount  uint32
	lastPoll     time.Time
	pollInterval time.Duration
}

func (f *liteFile) Close() error {
	f.logger.Info("closing file")
	return nil
}

func (f *liteFile) ReadAt(p []byte, off int64) (n int, err error) {
	f.logger.Info("reading at", "off", off, "len", len(p))

	pgno := uint32(1)
	if off >= 512 {
		pgno += uint32(off / int64(f.pageSize))
	}

	elem, ok := f.index[pgno]
	if !ok {
		f.logger.Error("page not found", "page", pgno)
		return 0, sqlite3.IOERR_READ
	}

	ctx := context.Background()
	if f.conn != nil {
		ctx = f.conn.GetInterrupt()
	}

	_, data, err := litestream.FetchPage(ctx, f.client, elem.Level, elem.MinTXID, elem.MaxTXID, elem.Offset, elem.Size)
	if err != nil {
		f.logger.Error("cannot fetch page", "error", err)
		return 0, sqlite3.IOERR_READ
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
	f.logger.Info("data read", "n", n, "data", len(data))

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
	f.logger.Info("file size", "size", size)
	return size, nil
}

func (f *liteFile) Lock(lock vfs.LockLevel) error {
	f.logger.Info("locking file", "lock", lock)
	if lock >= vfs.LOCK_RESERVED {
		return sqlite3.IOERR_LOCK
	}
	return f.pollReplicaClient()
}

func (f *liteFile) Unlock(lock vfs.LockLevel) error {
	f.logger.Info("unlocking file", "lock", lock)
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

	f.logger.Debug("polling replica client", "txid", f.pos.TXID.String())

	// Start reading from the next LTX file after the current position.
	itr, err := f.client.LTXFiles(ctx, 0, f.pos.TXID+1)
	if err != nil {
		return fmt.Errorf("ltx files: %w", err)
	}
	if !itr.Next() {
		return nil // No changes.
	}

	err = f.buildIndex(ctx)
	if err == nil {
		f.changeCount++
	}
	return err
}

func (f *liteFile) buildIndex(ctx context.Context) error {
	infos, err := litestream.CalcRestorePlan(ctx, f.client, 0, time.Time{}, f.logger)
	if err != nil {
		return fmt.Errorf("cannot calc restore plan: %w", err)
	} else if len(infos) == 0 {
		return fmt.Errorf("no backup files available") // TODO: Open even when no files available.
	}

	// Determine the current position based off the latest LTX file.
	if len(infos) > 0 {
		f.pos = ltx.Pos{TXID: infos[len(infos)-1].MaxTXID}
	}

	clear(f.index)
	for _, info := range infos {
		f.logger.Debug("opening page index", "level", info.Level, "min", info.MinTXID, "max", info.MaxTXID)

		// Read page index.
		idx, err := litestream.FetchPageIndex(ctx, f.client, info)
		if err != nil {
			return fmt.Errorf("fetch page index: %w", err)
		}

		// Replace pages in overall index with new pages.
		for k, v := range idx {
			f.logger.Debug("adding page index", "page", k, "elem", v)
			f.pageCount = max(f.pageCount, k)
			f.index[k] = v
		}
	}
	return nil
}
