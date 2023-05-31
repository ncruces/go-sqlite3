package sqlite3memdb

import (
	"io"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/sqlite3vfs"
)

type vfs struct{}

func (vfs) Open(name string, flags sqlite3vfs.OpenFlag) (sqlite3vfs.File, sqlite3vfs.OpenFlag, error) {
	if flags&sqlite3vfs.OPEN_MAIN_DB == 0 {
		return nil, flags, sqlite3.CANTOPEN
	}

	var db *dbase

	shared := strings.HasPrefix(name, "/")
	if shared {
		memoryMtx.Lock()
		defer memoryMtx.Unlock()
		db = memoryDBs[name[1:]]
	}
	if db == nil {
		if flags&sqlite3vfs.OPEN_CREATE == 0 {
			return nil, flags, sqlite3.CANTOPEN
		}
		db = new(dbase)
	}
	if shared {
		memoryDBs[name[1:]] = db
	}

	return &file{
		dbase:    db,
		readOnly: flags&sqlite3vfs.OPEN_READONLY != 0,
	}, flags | sqlite3vfs.OPEN_MEMORY, nil
}

func (vfs) Delete(name string, dirSync bool) error {
	return sqlite3.IOERR_DELETE
}

func (vfs) Access(name string, flag sqlite3vfs.AccessFlag) (bool, error) {
	return false, nil
}

func (vfs) FullPathname(name string) (string, error) {
	return name, nil
}

const sectorSize = 65536

type dbase struct {
	// +checklocks:lockMtx
	pending *file
	// +checklocks:lockMtx
	reserved *file

	// +checklocks:dataMtx
	data []*[sectorSize]byte

	// +checklocks:dataMtx
	size int64

	// +checklocks:lockMtx
	shared int

	lockMtx sync.Mutex
	dataMtx sync.RWMutex
}

type file struct {
	*dbase
	lock     sqlite3vfs.LockLevel
	readOnly bool
}

var (
	// Ensure these interfaces are implemented:
	_ sqlite3vfs.FileLockState = &file{}
	_ sqlite3vfs.FileSizeHint  = &file{}
)

func (m *file) Close() error {
	return m.Unlock(sqlite3vfs.LOCK_NONE)
}

func (m *file) ReadAt(b []byte, off int64) (n int, err error) {
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

func (m *file) WriteAt(b []byte, off int64) (n int, err error) {
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

func (m *file) Truncate(size int64) error {
	m.dataMtx.Lock()
	defer m.dataMtx.Unlock()
	return m.truncate(size)
}

// +checklocks:m.dataMtx
func (m *file) truncate(size int64) error {
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

func (*file) Sync(flag sqlite3vfs.SyncFlag) error {
	return nil
}

func (m *file) Size() (int64, error) {
	m.dataMtx.RLock()
	defer m.dataMtx.RUnlock()
	return m.size, nil
}

func (m *file) Lock(lock sqlite3vfs.LockLevel) error {
	if m.lock >= lock {
		return nil
	}

	if m.readOnly && lock >= sqlite3vfs.LOCK_RESERVED {
		return sqlite3.IOERR_LOCK
	}

	m.lockMtx.Lock()
	defer m.lockMtx.Unlock()
	deadline := time.Now().Add(time.Millisecond)

	switch lock {
	case sqlite3vfs.LOCK_SHARED:
		for m.pending != nil {
			if time.Now().After(deadline) {
				return sqlite3.BUSY
			}
			m.lockMtx.Unlock()
			runtime.Gosched()
			m.lockMtx.Lock()
		}
		m.shared++

	case sqlite3vfs.LOCK_RESERVED:
		if m.reserved != nil {
			return sqlite3.BUSY
		}
		m.reserved = m

	case sqlite3vfs.LOCK_EXCLUSIVE:
		if m.lock < sqlite3vfs.LOCK_PENDING {
			if m.pending != nil {
				return sqlite3.BUSY
			}
			m.lock = sqlite3vfs.LOCK_PENDING
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

func (m *file) Unlock(lock sqlite3vfs.LockLevel) error {
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
	if lock < sqlite3vfs.LOCK_SHARED {
		m.shared--
	}
	m.lock = lock
	return nil
}

func (m *file) CheckReservedLock() (bool, error) {
	if m.lock >= sqlite3vfs.LOCK_RESERVED {
		return true, nil
	}
	m.lockMtx.Lock()
	defer m.lockMtx.Unlock()
	return m.reserved != nil, nil
}

func (*file) SectorSize() int {
	return sectorSize
}

func (*file) DeviceCharacteristics() sqlite3vfs.DeviceCharacteristic {
	return sqlite3vfs.IOCAP_ATOMIC |
		sqlite3vfs.IOCAP_SEQUENTIAL |
		sqlite3vfs.IOCAP_SAFE_APPEND |
		sqlite3vfs.IOCAP_POWERSAFE_OVERWRITE
}

func (m *file) SizeHint(size int64) error {
	m.dataMtx.Lock()
	defer m.dataMtx.Unlock()
	if size > m.size {
		return m.truncate(size)
	}
	return nil
}

func (m *file) LockState() sqlite3vfs.LockLevel {
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
