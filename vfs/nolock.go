//go:build sqlite3_nolock

package vfs

const (
	_PENDING_BYTE  = 0x40000000
	_RESERVED_BYTE = (_PENDING_BYTE + 1)
	_SHARED_FIRST  = (_PENDING_BYTE + 2)
	_SHARED_SIZE   = 510
)

func (f *vfsFile) Lock(lock LockLevel) error {
	return nil
}

func (f *vfsFile) Unlock(lock LockLevel) error {
	return nil
}

func (f *vfsFile) CheckReservedLock() (bool, error) {
	return false, nil
}
