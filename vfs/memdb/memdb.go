package memdb

import (
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/vfs"
)

type memVFS struct{}

func (memVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// Allowed file types:
	// - databases, which only do page aligned reads/writes;
	// - temp journals, used by the sorter, which does the same.
	const types = vfs.OPEN_MAIN_DB |
		vfs.OPEN_TRANSIENT_DB |
		vfs.OPEN_TEMP_DB |
		vfs.OPEN_TEMP_JOURNAL
	if flags&types == 0 {
		return nil, flags, sqlite3.CANTOPEN
	}

	var db *memDB

	shared := strings.HasPrefix(name, "/")
	if shared {
		memoryMtx.Lock()
		defer memoryMtx.Unlock()
		db = memoryDBs[name[1:]]
	}
	if db == nil {
		if flags&vfs.OPEN_CREATE == 0 {
			return nil, flags, sqlite3.CANTOPEN
		}
		db = new(memDB)
	}
	if shared {
		memoryDBs[name[1:]] = db
	}

	return &memFile{
		memDB:    db,
		readOnly: flags&vfs.OPEN_READONLY != 0,
	}, flags | vfs.OPEN_MEMORY, nil
}

func (memVFS) Delete(name string, dirSync bool) error {
	return sqlite3.IOERR_DELETE
}

func (memVFS) Access(name string, flag vfs.AccessFlag) (bool, error) {
	return false, nil
}

func (memVFS) FullPathname(name string) (string, error) {
	return name, nil
}

// Must be a multiple of 64K (the largest page size).
const sectorSize = 65536

type memDB struct {
	// +checklocks:lockMtx
	pending *memFile
	// +checklocks:lockMtx
	reserved *memFile

	// +checklocks:dataMtx
	data []*[sectorSize]byte

	// +checklocks:dataMtx
	size int64

	// +checklocks:lockMtx
	shared int

	lockMtx sync.Mutex
	dataMtx sync.RWMutex
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
	if base == int64(len(m.data))-1 {
		have = modRoundUp(m.size, sectorSize)
	}
	n = copy(b, (*m.data[base])[rest:have])
	if n < len(b) {
		// Assume reads are page aligned.
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
	if n < len(b) {
		// Assume writes are page aligned.
		return 0, io.ErrShortWrite
	}
	if size := off + int64(len(b)); size > m.size {
		m.size = size
	}
	return n, nil
}

func (m *memFile) Truncate(size int64) error {
	m.dataMtx.Lock()
	defer m.dataMtx.Unlock()
	return m.truncate(size)
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

func (*memFile) Sync(flag vfs.SyncFlag) error {
	return nil
}

func (m *memFile) Size() (int64, error) {
	m.dataMtx.RLock()
	defer m.dataMtx.RUnlock()
	return m.size, nil
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
	deadline := time.Now().Add(time.Millisecond)

	switch lock {
	case vfs.LOCK_SHARED:
		for m.pending != nil {
			if time.Now().After(deadline) {
				return sqlite3.BUSY
			}
			m.lockMtx.Unlock()
			runtime.Gosched()
			m.lockMtx.Lock()
		}
		m.shared++

	case vfs.LOCK_RESERVED:
		if m.reserved != nil {
			return sqlite3.BUSY
		}
		m.reserved = m

	case vfs.LOCK_EXCLUSIVE:
		if m.lock < vfs.LOCK_PENDING {
			if m.pending != nil {
				return sqlite3.BUSY
			}
			m.lock = vfs.LOCK_PENDING
			m.pending = m
		}

		for m.shared > 1 {
			if time.Now().After(deadline) {
				return sqlite3.BUSY
			}
			m.lockMtx.Unlock()
			runtime.Gosched()
			m.lockMtx.Lock()
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

	if m.pending == m {
		m.pending = nil
	}
	if m.reserved == m {
		m.reserved = nil
	}
	if lock < vfs.LOCK_SHARED {
		m.shared--
	}
	m.lock = lock
	return nil
}

func (m *memFile) CheckReservedLock() (bool, error) {
	if m.lock >= vfs.LOCK_RESERVED {
		return true, nil
	}
	m.lockMtx.Lock()
	defer m.lockMtx.Unlock()
	return m.reserved != nil, nil
}

func (*memFile) SectorSize() int {
	return sectorSize
}

func (*memFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_ATOMIC |
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_SAFE_APPEND |
		vfs.IOCAP_POWERSAFE_OVERWRITE
}

func (m *memFile) SizeHint(size int64) error {
	m.dataMtx.Lock()
	defer m.dataMtx.Unlock()
	if size > m.size {
		return m.truncate(size)
	}
	return nil
}

func (m *memFile) LockState() vfs.LockLevel {
	return m.lock
}

func divRoundUp(a, b int64) int64 {
	return (a + b - 1) / b
}

func modRoundUp(a, b int64) int64 {
	return b - (b-a%b)%b
}

func clear[T any](b []T) {
	var zero T
	for i := range b {
		b[i] = zero
	}
}
