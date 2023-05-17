package sqlite3vfs

import "sync"

type VFS interface {
	Open(name string, flags OpenFlag) (File, OpenFlag, error)
	Delete(name string, dirSync bool) error
	Access(name string, flags AccessFlag) (bool, error)
	FullPathname(name string) (string, error)
}

type File interface {
	Close() error
	ReadAt(p []byte, off int64) (n int, err error)
	WriteAt(p []byte, off int64) (n int, err error)
	Truncate(size int64) error
	Sync(flag SyncFlag) error
	FileSize() (int64, error)
	Lock(elock LockLevel) error
	Unlock(elock LockLevel) error
	CheckReservedLock() (bool, error)
	SectorSize() int64
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
