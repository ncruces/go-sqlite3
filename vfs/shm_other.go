//go:build !(linux || darwin) || !(amd64 || arm64) || sqlite3_flock || sqlite3_nosys

package vfs

import (
	"context"

	"github.com/tetratelabs/wazero/api"
)

// SupportsSharedMemory is true on platforms that support shared memory.
// To enable shared memory support on those platforms,
// you need to set the appropriate [wazero.RuntimeConfig];
// otherwise, [EXCLUSIVE locking mode] is activated automatically
// to use [WAL without shared-memory].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = false

type vfsShm struct{}

func (f *vfsFile) ShmMap(ctx context.Context, mod api.Module, id, size uint32, extend bool) (uint32, error) {
	return 0, _IOERR_SHMMAP
}

func (f *vfsFile) ShmLock(offset, n uint32, flags _ShmFlag) error {
	return _IOERR_SHMLOCK
}

func (f *vfsFile) ShmUnmap(delete bool) {}
