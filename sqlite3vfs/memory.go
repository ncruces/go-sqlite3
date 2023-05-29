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

// A MemoryDB is a [MemoryVFS] database.
//
// A MemoryDB is safe to access concurrently from multiple SQLite connections.
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

// Close implements the [File] and [io.Closer] interfaces.
func (m *memoryFile) Close() error {
	return m.Unlock(LOCK_NONE)
}

// ReadAt implements the [File] and [io.ReaderAt] interfaces.
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
		have = modRoundUp(m.size, memSectorSize)
	}
	n = copy(b, (*m.data[base])[rest:have])
	if n < len(b) {
		// Assume reads are page aligned.
		return 0, io.ErrNoProgress
	}
	return n, nil
}

// WriteAt implements the [File] and [io.WriterAt] interfaces.
func (m *memoryFile) WriteAt(b []byte, off int64) (n int, err error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	base := off / memSectorSize
	rest := off % memSectorSize
	for base >= int64(len(m.data)) {
		m.data = append(m.data, new([memSectorSize]byte))
	}
	n = copy((*m.data[base])[rest:], b)
	if n < len(b) {
		// Assume writes are page aligned.
		return 0, io.ErrShortWrite
	}
	return n, nil
}

// Truncate implements the [File] interface.
func (m *memoryFile) Truncate(size int64) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.truncate(size)
}

func (m *memoryFile) truncate(size int64) error {
	if size < m.size {
		base := size / memSectorSize
		rest := size % memSectorSize
		if rest != 0 {
			clear((*m.data[base])[rest:])
		}
	}
	sectors := divRoundUp(size, memSectorSize)
	for sectors > int64(len(m.data)) {
		m.data = append(m.data, new([memSectorSize]byte))
	}
	clear(m.data[sectors:])
	m.data = m.data[:sectors]
	m.size = size
	return nil
}

// Sync implements the [File] interface.
func (*memoryFile) Sync(flag SyncFlag) error {
	return nil
}

// Size implements the [File] interface.
func (m *memoryFile) Size() (int64, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.size, nil
}

// Lock implements the [File] interface.
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

// Unlock implements the [File] interface.
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

// CheckReservedLock implements the [File] interface.
func (m *memoryFile) CheckReservedLock() (bool, error) {
	if m.lock >= LOCK_RESERVED {
		return true, nil
	}
	m.locker.Lock()
	defer m.locker.Unlock()
	return m.reserved != nil, nil
}

// SectorSize implements the [File] interface.
func (*memoryFile) SectorSize() int {
	return memSectorSize
}

// DeviceCharacteristics implements the [File] interface.
func (*memoryFile) DeviceCharacteristics() DeviceCharacteristic {
	return IOCAP_ATOMIC |
		IOCAP_SEQUENTIAL |
		IOCAP_SAFE_APPEND |
		IOCAP_POWERSAFE_OVERWRITE
}

// SizeHint implements the [FileSizeHint] interface.
func (m *memoryFile) SizeHint(size int64) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if size > m.size {
		return m.truncate(size)
	}
	return nil
}

// LockState implements the [FileLockState] interface.
func (m *memoryFile) LockState() LockLevel {
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
