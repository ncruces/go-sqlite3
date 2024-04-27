//go:build !(darwin || linux) || !(amd64 || arm64 || riscv64) || sqlite3_flock || sqlite3_noshm || sqlite3_nosys

package vfs

// SupportsSharedMemory is true on platforms that support shared memory.
// To enable shared memory support on those platforms,
// you need to set the appropriate [wazero.RuntimeConfig];
// otherwise, [EXCLUSIVE locking mode] is activated automatically
// to use [WAL without shared-memory].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = false

// NewSharedMemory returns a shared-memory WAL-index
// backed a file with the given path.
// It may return nil if shared-memory is not supported,
// or not appropriate for the given flags.
// Only [OPEN_MAIN_DB] databases support WAL mode.
func NewSharedMemory(path string, flags OpenFlag) SharedMemory {
	return nil
}
