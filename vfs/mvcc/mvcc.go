package mvcc

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/aa"
	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

const sectorSize = 512

type mvccVFS struct{}

func (mvccVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// For simplicity, we do not support reading or writing data
	// across "sector" boundaries.
	// This is not a problem for SQLite database files.
	const databases = vfs.OPEN_MAIN_DB | vfs.OPEN_TEMP_DB | vfs.OPEN_TRANSIENT_DB

	// Temp journals, as used by the sorter, use SliceFile.
	if flags&vfs.OPEN_TEMP_JOURNAL != 0 {
		return &vfsutil.SliceFile{}, flags | vfs.OPEN_MEMORY, nil
	}

	// Refuse to open all other file types.
	// Returning OPEN_MEMORY means SQLite won't ask us to.
	if flags&databases == 0 {
		// notest // OPEN_MEMORY
		return nil, flags, sqlite3.CANTOPEN
	}

	// A shared database has a name that begins with "/".
	shared := strings.HasPrefix(name, "/")

	var db *mvccDB
	if shared {
		name = name[1:]
		memoryMtx.Lock()
		defer memoryMtx.Unlock()
		db = memoryDBs[name]
	}
	if db == nil {
		if flags&vfs.OPEN_CREATE == 0 {
			return nil, flags, sqlite3.CANTOPEN
		}
		db = &mvccDB{name: name}
	}
	if shared {
		db.refs++ // +checklocksforce: memoryMtx is held
		memoryDBs[name] = db
	}

	return &mvccFile{
		mvccDB:   db,
		readOnly: flags&vfs.OPEN_READONLY != 0,
	}, flags | vfs.OPEN_MEMORY, nil
}

func (mvccVFS) Delete(name string, dirSync bool) error {
	return sqlite3.IOERR_DELETE_NOENT // used to delete journals
}

func (mvccVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	return false, nil // used to check for journals
}

func (mvccVFS) FullPathname(name string) (string, error) {
	return name, nil
}

type mvccDB struct {
	data   *aa.Tree[int64, []byte] // +checklocks:mtx
	owner  *mvccFile               // +checklocks:mtx
	waiter *sync.Cond              // +checklocks:mtx

	name string
	refs int   // +checklocks:memoryMtx
	size int64 // +checklocks:mtx
	mtx  sync.Mutex
}

func (m *mvccDB) release() {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	if m.refs--; m.refs == 0 && m == memoryDBs[m.name] {
		delete(memoryDBs, m.name)
	}
}

func (m *mvccDB) fork() *mvccDB {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return &mvccDB{
		refs: 1,
		name: m.name,
		data: m.data,
		size: m.size,
	}
}

type mvccFile struct {
	*mvccDB
	data     *aa.Tree[int64, []byte]
	size     int64
	lock     vfs.LockLevel
	readOnly bool
}

var (
	// Ensure these interfaces are implemented:
	_ vfs.FileLockState = &mvccFile{}
)

func (m *mvccFile) Close() error {
	// Relase ownership, discard changes.
	m.release()
	m.data = nil
	m.size = 0
	m.lock = vfs.LOCK_NONE
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.owner == m {
		m.owner = nil
	}
	return nil
}

func (m *mvccFile) ReadAt(b []byte, off int64) (n int, err error) {
	data := m.data
	size := m.size
	if data == nil {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		data = m.mvccDB.data
		size = m.mvccDB.size
	}

	have := size - off
	if have <= 0 {
		return 0, io.EOF
	}

	base := off / sectorSize
	rest := off % sectorSize

	v, _ := data.Get(base)
	v = v[rest:]
	v = v[:min(have, int64(len(v)))]

	n = copy(b, v)
	if n < len(b) {
		// notest // assume reads are page aligned
		return 0, io.ErrNoProgress
	}
	return n, nil
}

func (m *mvccFile) WriteAt(b []byte, off int64) (n int, err error) {
	if len(b)%sectorSize != 0 || off%sectorSize != 0 {
		// notest // assume writes are page aligned
		return 0, io.ErrShortWrite
	}
	data := m.data
	base := off / sectorSize
	for i := 1; i*sectorSize < len(b); i++ {
		data = data.Delete(base + int64(i))
	}
	m.data = data.Put(base, append([]byte(nil), b...))
	if size := off + int64(len(b)); size > m.size {
		m.size = size
	}
	return len(b), nil
}

func (m *mvccFile) Size() (int64, error) {
	data := m.data
	size := m.size
	if data == nil {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		size = m.mvccDB.size
	}
	return size, nil
}

func (m *mvccFile) Truncate(size int64) error {
	var i int64
	for data := m.data; data != nil; data = data.Right() {
		i = data.Key()
	}
	for ; i*sectorSize >= size; i-- {
		m.data = m.data.Delete(i)
	}
	m.size = size
	return nil
}

func (m *mvccFile) Lock(lock vfs.LockLevel) error {
	if m.lock >= lock {
		return nil
	}

	if m.readOnly && lock >= vfs.LOCK_RESERVED {
		return sqlite3.IOERR_LOCK
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Take a snapshot of the database.
	if lock == vfs.LOCK_SHARED {
		m.data = m.mvccDB.data
		m.size = m.mvccDB.size
		m.lock = lock
		return nil
	}
	// We are the owners.
	if m.owner == m {
		m.lock = lock
		return nil
	}
	// Someone else is the owner.
	if m.owner != nil {
		before := time.Now()
		if m.waiter == nil {
			m.waiter = sync.NewCond(&m.mtx)
		}
		defer time.AfterFunc(time.Millisecond, m.waiter.Broadcast).Stop()
		for m.owner != nil {
			// Our snapshot is invalid.
			if m.data != m.mvccDB.data || m.size != m.mvccDB.size {
				return sqlite3.BUSY_SNAPSHOT
			}
			if time.Since(before) > time.Millisecond {
				return sqlite3.BUSY
			}
			m.waiter.Wait()
		}
	}
	// Our snapshot is invalid.
	if m.data != m.mvccDB.data || m.size != m.mvccDB.size {
		return sqlite3.BUSY_SNAPSHOT
	}
	// Take ownership.
	m.lock = lock
	m.owner = m
	return nil
}

func (m *mvccFile) Unlock(lock vfs.LockLevel) error {
	if m.lock <= lock {
		return nil
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Relase ownership, commit changes.
	if m.owner == m {
		m.owner = nil
		m.mvccDB.data = m.data
		m.mvccDB.size = m.size
		if m.waiter != nil {
			m.waiter.Broadcast()
		}
	}
	m.lock = lock
	return nil
}

func (m *mvccFile) CheckReservedLock() (bool, error) {
	// notest // OPEN_MEMORY
	if m.lock >= vfs.LOCK_RESERVED {
		return true, nil
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.owner != nil, nil
}

func (m *mvccFile) LockState() vfs.LockLevel {
	return m.lock
}

func (*mvccFile) Sync(flag vfs.SyncFlag) error { return nil }

func (*mvccFile) SectorSize() int {
	// notest // IOCAP_POWERSAFE_OVERWRITE
	return sectorSize
}

func (*mvccFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_ATOMIC |
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_SAFE_APPEND |
		vfs.IOCAP_POWERSAFE_OVERWRITE
}

func divRoundUp(a, b int64) int64 {
	return (a + b - 1) / b
}
