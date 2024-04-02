//go:build !(linux || darwin) || !(amd64 || arm64) || sqlite3_nosys

package vfs

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

type vfsShm struct{}

func (f *vfsFile) ShmMap(ctx context.Context, mod api.Module, id, size uint32, extend bool) (uint32, error) {
	return 0, _IOERR_SHMMAP
}

func (f *vfsFile) ShmLock(offset, n uint32, flags _ShmFlag) error {
	return _IOERR_SHMLOCK
}

func (f *vfsFile) ShmUnmap(delete bool) {}
