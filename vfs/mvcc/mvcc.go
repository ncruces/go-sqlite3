package mvcc

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/util/vfsutil"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/wbt"
)

type mvccVFS struct{}

func (mvccVFS) Open(name string, flags vfs.OpenFlag) (vfs.File, vfs.OpenFlag, error) {
	// Temporary files use SliceFile.
	if name == "" || flags&vfs.OPEN_DELETEONCLOSE != 0 {
		return &vfsutil.SliceFile{}, flags | vfs.OPEN_MEMORY, nil
	}

	// Only main databases benefit from multiversion concurrency control.
	// Refuse to open all other file types.
	// Returning OPEN_MEMORY means SQLite won't ask us to.
	if flags&vfs.OPEN_MAIN_DB == 0 {
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
	data   *wbt.Tree[int64, string] // +checklocks:mtx
	owner  *mvccFile                // +checklocks:mtx
	waiter *sync.Cond               // +checklocks:mtx

	name string
	refs int // +checklocks:memoryMtx
	mtx  sync.Mutex
}

func (m *mvccDB) release() {
	memoryMtx.Lock()
	defer memoryMtx.Unlock()
	if m.refs--; m.refs == 0 && m == memoryDBs[m.name] {
		delete(memoryDBs, m.name)
	}
}

type mvccFile struct {
	*mvccDB
	data     *wbt.Tree[int64, string]
	lock     vfs.LockLevel
	readOnly bool
}

var (
	// Ensure these interfaces are implemented:
	_ vfs.FileLockState      = &mvccFile{}
	_ vfs.FileCommitPhaseTwo = &mvccFile{}
)

func (m *mvccFile) Close() error {
	// Relase ownership, discard changes.
	m.release()
	m.data = nil
	m.lock = vfs.LOCK_NONE
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.owner == m {
		m.owner = nil
	}
	return nil
}

func (m *mvccFile) ReadAt(b []byte, off int64) (n int, err error) {
	// If unlocked, use a snapshot of the database.
	data := m.data
	if m.lock == vfs.LOCK_NONE {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		data = m.mvccDB.data
	}

	for k, v := range data.AscendFloor(off) {
		if i := k - off; i >= 0 {
			if +i > int64(n) {
				// Missing data.
				clear(b[n:])
			}
			if +i < int64(len(b)) {
				// Copy prefix.
				n = copy(b[+i:], v) + int(i)
			}
		} else {
			if -i < int64(len(v)) {
				// Copy suffix.
				n = copy(b, v[-i:])
			}
		}
		if n >= len(b) {
			return n, nil
		}
	}
	return n, io.EOF
}

func (m *mvccFile) WriteAt(b []byte, off int64) (n int, err error) {
	// If unlocked, take a snapshot of the database.
	data := m.data
	if m.lock == vfs.LOCK_NONE {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		data = m.mvccDB.data
		m.lock = vfs.LOCK_EXCLUSIVE + 1 // UNKNOWN_LOCK
	}

	next := off + int64(len(b))
	for k, v := range data.AscendFloor(off) {
		if k >= next {
			break
		}
		switch {
		case k > off:
			// Delete overlap.
			data = data.Delete(k)
		case k < off && off < k+int64(len(v)):
			// Reinsert prefix.
			data = data.Put(k, v[:off-k])
		}
		if k+int64(len(v)) > next {
			// Reinsert suffix.
			data = data.Put(next, v[next-k:])
		}
	}

	m.data = data.Put(off, string(b))
	return len(b), nil
}

func (m *mvccFile) Size() (int64, error) {
	// If unlocked, use a snapshot of the database.
	data := m.data
	if m.lock == vfs.LOCK_NONE {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		data = m.mvccDB.data
	}

	if data == nil {
		return 0, nil
	}
	data = data.Max()
	return data.Key() + int64(len(data.Value())), nil
}

func (m *mvccFile) Truncate(size int64) error {
	// If unlocked, take a snapshot of the database.
	data := m.data
	if m.lock == vfs.LOCK_NONE {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		data = m.mvccDB.data
		m.lock = vfs.LOCK_EXCLUSIVE + 1 // UNKNOWN_LOCK
	}

	for data != nil && data.Key() >= size {
		data = data.Left()
	}
	for k := range data.AscendCeil(size) {
		data = data.Delete(k)
	}
	m.data = data
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
			if m.data != m.mvccDB.data {
				return sqlite3.BUSY_SNAPSHOT
			}
			if time.Since(before) > time.Millisecond {
				return sqlite3.BUSY
			}
			m.waiter.Wait()
		}
	}
	// Our snapshot is invalid.
	if m.data != m.mvccDB.data {
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
		if m.waiter != nil {
			m.waiter.Broadcast()
		}
	}
	if lock == vfs.LOCK_NONE {
		m.data = nil
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

func (m *mvccFile) CommitPhaseTwo() error {
	// Modified without lock, commit changes.
	if m.lock > vfs.LOCK_EXCLUSIVE {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		m.mvccDB.data = m.data
		m.lock = vfs.LOCK_NONE
		m.data = nil
	}
	return nil
}

func (m *mvccFile) LockState() vfs.LockLevel {
	return m.lock
}

func (*mvccFile) Sync(flag vfs.SyncFlag) error { return nil }

func (*mvccFile) SectorSize() int {
	// notest // safe default
	return 0
}

func (*mvccFile) DeviceCharacteristics() vfs.DeviceCharacteristic {
	return vfs.IOCAP_ATOMIC |
		vfs.IOCAP_SEQUENTIAL |
		vfs.IOCAP_SAFE_APPEND |
		vfs.IOCAP_POWERSAFE_OVERWRITE |
		vfs.IOCAP_SUBPAGE_READ
}
