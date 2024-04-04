//go:build !(linux || darwin) || !(amd64 || arm64) || sqlite3_flock || sqlite3_nosys

package vfs

import "github.com/tetratelabs/wazero/api"

// SupportsSharedMemory is true on platforms that support shared memory.
// To enable shared memory support on those platforms,
// you need to set the appropriate [wazero.RuntimeConfig];
// otherwise, [EXCLUSIVE locking mode] is activated automatically
// to use [WAL without shared-memory].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = false

func vfsVersion(api.Module) uint32 { return 0 }

type vfsShm struct{}

func (vfsShm) Close() error { return nil }
