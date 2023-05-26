package sqlite3vfs

import (
	"io"
	"runtime"
	"sync"
	"time"
)

// A MemoryVFS is a [VFS] for memory databases.
type MemoryVFS map[string]*MemoryDB

var _ VFS = MemoryVFS{}

// Open implements the [VFS] interface.
func (vfs MemoryVFS) Open(name string, flags OpenFlag) (File, OpenFlag, error) {
	if flags&OPEN_MAIN_DB == 0 {
		return nil, flags, _CANTOPEN
	}
	if db, ok := vfs[name]; ok {
		return &memoryFile{
			MemoryDB: db,
			readOnly: flags&OPEN_READONLY != 0,
		}, flags, nil
	}
	return nil, flags, _CANTOPEN
}

// Delete implements the [VFS] interface.
func (vfs MemoryVFS) Delete(name string, dirSync bool) error {
	return _IOERR_DELETE
}

// Access implements the [VFS] interface.
func (vfs MemoryVFS) Access(name string, flag AccessFlag) (bool, error) {
	return false, nil
}

// FullPathname implements the [VFS] interface.
func (vfs MemoryVFS) FullPathname(name string) (string, error) {
	return name, nil
}

const memSectorSize = 65536

type MemoryDB struct {
	mtx  sync.RWMutex
	size int64
	data []*[memSectorSize]byte

	locker   sync.Mutex
	pending  *memoryFile
	reserved *memoryFile
	shared   int
}

type memoryFile struct {
	*MemoryDB
	lock     LockLevel
	readOnly bool
}

func (m *memoryFile) Close() error {
	return m.Unlock(LOCK_NONE)
}

func (m *memoryFile) ReadAt(b []byte, off int64) (n int, err error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	if off >= m.size {
		return 0, io.EOF
	}
	base := off / memSectorSize
	rest := off % memSectorSize
	have := int64(memSectorSize)
	if base == int64(len(m.data))-1 {
		have = m.size % memSectorSize
	}
	return copy(b, (*m.data[base])[rest:have]), nil
}

func (m *memoryFile) WriteAt(b []byte, off int64) (n int, err error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	base := off / memSectorSize
	rest := off % memSectorSize
	if base >= int64(len(m.data)) {
		m.data = append(m.data, new([memSectorSize]byte))
	}
	n = copy((*m.data[base])[rest:], b)
	if size := off + int64(n); size > m.size {
		m.size = size
	}
	return n, nil
}

func (m *memoryFile) Truncate(size int64) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.truncate(size)
}

func (m *memoryFile) truncate(size int64) error {
	if size < m.size {
		base := size / memSectorSize
		rest := size % memSectorSize
		clear((*m.data[base])[rest:])
	}
	sectors := (size + memSectorSize - 1) / memSectorSize
	for sectors > int64(len(m.data)) {
		m.data = append(m.data, new([memSectorSize]byte))
	}
	for sectors < int64(len(m.data)) {
		last := int64(len(m.data)) - 1
		m.data[last] = nil
		m.data = m.data[:last]
	}
	m.size = size
	return nil
}

func (*memoryFile) Sync(flag SyncFlag) error {
	return nil
}

func (m *memoryFile) Size() (int64, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.size, nil
}

func (m *memoryFile) Lock(lock LockLevel) error {
	if m.lock >= lock {
		return nil
	}

	if m.readOnly && lock >= LOCK_RESERVED {
		return _IOERR_LOCK
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	deadline := time.Now().Add(time.Millisecond)

	switch lock {
	case LOCK_SHARED:
		for m.pending != nil {
			if time.Now().After(deadline) {
				return _BUSY
			}
			m.locker.Unlock()
			runtime.Gosched()
			m.locker.Lock()
		}
		m.shared++

	case LOCK_RESERVED:
		if m.reserved != nil {
			return _BUSY
		}
		m.reserved = m

	case LOCK_EXCLUSIVE:
		if m.lock < LOCK_PENDING {
			if m.pending != nil {
				return _BUSY
			}
			m.lock = LOCK_PENDING
			m.pending = m
		}

		for m.shared > 1 {
			if time.Now().After(deadline) {
				return _BUSY
			}
			m.locker.Unlock()
			runtime.Gosched()
			m.locker.Lock()
		}
	}

	m.lock = lock
	return nil
}

func (m *memoryFile) Unlock(lock LockLevel) error {
	if m.lock <= lock {
		return nil
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	if m.pending == m {
		m.pending = nil
	}
	if m.reserved == m {
		m.reserved = nil
	}
	if lock < LOCK_SHARED {
		m.shared--
	}
	m.lock = lock
	return nil
}

func (m *memoryFile) CheckReservedLock() (bool, error) {
	if m.lock >= LOCK_RESERVED {
		return true, nil
	}
	m.locker.Lock()
	defer m.locker.Unlock()
	return m.reserved != nil, nil
}

func (*memoryFile) SectorSize() int {
	return memSectorSize
}

func (*memoryFile) DeviceCharacteristics() DeviceCharacteristic {
	return IOCAP_ATOMIC |
		IOCAP_SEQUENTIAL |
		IOCAP_SAFE_APPEND |
		IOCAP_POWERSAFE_OVERWRITE
}

func (m *memoryFile) SizeHint(size int64) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if size > m.size {
		return m.truncate(size)
	}
	return nil
}

func (m *memoryFile) LockState() LockLevel {
	return m.lock
}

func clear(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
