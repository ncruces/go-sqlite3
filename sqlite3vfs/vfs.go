package sqlite3vfs

import "sync"

type VFS interface {
	Open(name string, flags OpenFlag) (File, OpenFlag, error)
	Delete(name string, syncDir bool) error
	Access(name string, flags AccessFlag) (bool, error)
	FullPathname(name string) (string, error)
}

type File interface {
	Close() error
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
	Truncate(size int64) error
	Sync(flags SyncFlag) error
	FileSize() (int64, error)
	Lock(lock LockLevel) error
	Unlock(lock LockLevel) error
	CheckReservedLock() (bool, error)
	SectorSize() int
	DeviceCharacteristics() DeviceCharacteristic
}

type FileLockState interface {
	File
	LockState() LockLevel
}

type FileSizeHint interface {
	File
	SizeHint(size int64) error
}

type FileHasMoved interface {
	File
	HasMoved() (bool, error)
}

type FilePowersafeOverwrite interface {
	File
	PowersafeOverwrite() bool
	SetPowersafeOverwrite(bool)
}

var (
	vfsRegistry    map[string]VFS
	vfsRegistryMtx sync.Mutex
)

func Find(name string) VFS {
	vfsRegistryMtx.Lock()
	defer vfsRegistryMtx.Unlock()
	return vfsRegistry[name]
}

func Register(name string, vfs VFS) {
	vfsRegistryMtx.Lock()
	defer vfsRegistryMtx.Unlock()
	if vfsRegistry == nil {
		vfsRegistry = map[string]VFS{}
	}
	vfsRegistry[name] = vfs
}

func Unregister(name string) {
	vfsRegistryMtx.Lock()
	defer vfsRegistryMtx.Unlock()
	delete(vfsRegistry, name)
}
