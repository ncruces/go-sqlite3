package memdb

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

const sectorSize = 65536

type memVFS struct{}

func (memVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
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

	var db *memDB
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
		db = &memDB{name: name}
	}
	if shared {
		db.refs++ // +checklocksforce: memoryMtx is held
		memoryDBs[name] = db
	}

	return &memFile{
		memDB:    db,
		readOnly: flags&vfs.OPEN_READONLY != 0,
	}, flags | vfs.OPEN_MEMORY, nil
}

func (memVFS) Delete(name string, dirSync bool) error {
	return sqlite3.IOERR_DELETE_NOENT // used to delete journals
}

func (memVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	return false, nil // used to check for journals
}

func (memVFS) FullPathname(name string) (string, error) {
	return name, nil
}

type memDB struct {
	name string

	// +checklocks:lockMtx
	waiter *sync.Cond
	// +checklocks:dataMtx
	data []*[sectorSize]byte

	size     int64 // +checklocks:dataMtx
	refs     int32 // +checklocks:memoryMtx
	shared   int32 // +checklocks:lockMtx
	pending  bool  // +checklocks:lockMtx
	reserved bool  // +checklocks:lockMtx

	lockMtx sync.Mutex
	dataMtx sync.RWMutex
}

func (m *memDB) release() {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	if m.refs--; m.refs == 0 && m == memoryDBs[m.name] {
		delete(memoryDBs, m.name)
	}
}

type memFile struct {
	*memDB
	lock     vfs.LockLevel
	readOnly bool
}

var (
	// Ensure these interfaces are implemented:
	_ vfs.FileLockState = &memFile{}
	_ vfs.FileSizeHint  = &memFile{}
)

func (m *memFile) Close() error {
	m.release()
	return m.Unlock(vfs.LOCK_NONE)
}

func (m *memFile) ReadAt(b []byte, off int64) (n int, err error) {
	m.dataMtx.RLock()
	defer m.dataMtx.RUnlock()

	if off >= m.size {
		return 0, io.EOF
	}

	base := off / sectorSize
	rest := off % sectorSize
	have := int64(sectorSize)
	if m.size < off+int64(len(b)) {
		have = modRoundUp(m.size, sectorSize)
	}
	n = copy(b, (*m.data[base])[rest:have])
	if n < len(b) {
		// notest // assume reads are page aligned
		return 0, io.ErrNoProgress
	}
	return n, nil
}

func (m *memFile) WriteAt(b []byte, off int64) (n int, err error) {
	m.dataMtx.Lock()
	defer m.dataMtx.Unlock()

	base := off / sectorSize
	rest := off % sectorSize
	for base >= int64(len(m.data)) {
		m.data = append(m.data, new([sectorSize]byte))
	}
	n = copy((*m.data[base])[rest:], b)
	if size := off + int64(n); size > m.size {
		m.size = size
	}
	if n < len(b) {
		// notest // assume writes are page aligned
		return n, io.ErrShortWrite
	}
	return n, nil
}

func (m *memFile) Size() (int64, error) {
	m.dataMtx.RLock()
	defer m.dataMtx.RUnlock()
	return m.size, nil
}

func (m *memFile) Truncate(size int64) error {
	m.dataMtx.Lock()
	defer m.dataMtx.Unlock()
	return m.truncate(size)
}

func (m *memFile) SizeHint(size int64) error {
	m.dataMtx.Lock()
	defer m.dataMtx.Unlock()
	if size > m.size {
		return m.truncate(size)
	}
	return nil
}

// +checklocks:m.dataMtx
func (m *memFile) truncate(size int64) error {
	if size < m.size {
		base := size / sectorSize
		rest := size % sectorSize
		if rest != 0 {
			clear((*m.data[base])[rest:])
		}
	}
	sectors := divRoundUp(size, sectorSize)
	for sectors > int64(len(m.data)) {
		m.data = append(m.data, new([sectorSize]byte))
	}
	clear(m.data[sectors:])
	m.data = m.data[:sectors]
	m.size = size
	return nil
}

func (m *memFile) Lock(lock vfs.LockLevel) error {
	if m.lock >= lock {
		return nil
	}

	if m.readOnly && lock >= vfs.LOCK_RESERVED {
		return sqlite3.IOERR_LOCK
	}

	m.lockMtx.Lock()
	defer m.lockMtx.Unlock()

	switch lock {
	case vfs.LOCK_SHARED:
		if m.pending {
			return sqlite3.BUSY
		}
		m.shared++

	case vfs.LOCK_RESERVED:
		if m.reserved {
			return sqlite3.BUSY
		}
		m.reserved = true

	case vfs.LOCK_EXCLUSIVE:
		if m.lock < vfs.LOCK_PENDING {
			m.lock = vfs.LOCK_PENDING
			m.pending = true
		}

		if m.shared > 1 {
			before := time.Now()
			if m.waiter == nil {
				m.waiter = sync.NewCond(&m.lockMtx)
			}
			defer time.AfterFunc(time.Millisecond, m.waiter.Broadcast).Stop()
			for m.shared > 1 {
				if time.Since(before) > time.Millisecond {
					return sqlite3.BUSY
				}
				m.waiter.Wait()
			}
		}
	}

	m.lock = lock
	return nil
}

func (m *memFile) Unlock(lock vfs.LockLevel) error {
	if m.lock <= lock {
		return nil
	}

	m.lockMtx.Lock()
	defer m.lockMtx.Unlock()

	if m.lock >= vfs.LOCK_RESERVED {
		m.reserved = false
	}
	if m.lock >= vfs.LOCK_PENDING {
		m.pending = false
	}
	if lock < vfs.LOCK_SHARED {
		if m.shared--; m.pending && m.shared <= 1 && m.waiter != nil {
			m.waiter.Broadcast()
		}
	}
	m.lock = lock
	return nil
}

func (m *memFile) CheckReservedLock() (bool, error) {
	// notest // OPEN_MEMORY
	if m.lock >= vfs.LOCK_RESERVED {
		return true, nil
	}
	m.lockMtx.Lock()
	defer m.lockMtx.Unlock()
	return m.reserved, nil
}

func (m *memFile) LockState() vfs.LockLevel {
	return m.lock
}

func (*memFile) Sync(flag vfs.SyncFlag) error { return nil }

func (*memFile) SectorSize() int {
	// notest // IOCAP_POWERSAFE_OVERWRITE
	return sectorSize
}

func (*memFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_ATOMIC |
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_SAFE_APPEND |
		vfs.IOCAP_POWERSAFE_OVERWRITE
}

func divRoundUp(a, b int64) int64 {
	return (a + b - 1) / b
}

func modRoundUp(a, b int64) int64 {
	return b - (b-a%b)%b
}
