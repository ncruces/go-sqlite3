// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"context"
	"crypto/rand"
	"strconv"
	"time"
	_ "unsafe"

	sqlite3_wasm "github.com/ncruces/go-sqlite3-wasm"
	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/ncruces/julianday"
)

type configKey struct{}

// WithMaxMemory returns a derived context that configures
// each SQLite connection not to use more than max amount of memory.
func WithMaxMemory(ctx context.Context, max int64) context.Context {
	if max < 0 || max > 65536*65536 {
		panic(errutil.OOMErr)
	}
	return context.WithValue(ctx, configKey{}, max/65536)
}

var _ sqlite3_wasm.Xenv = &env{}

type env struct{ *sqlite3_wrap.Wrapper }

func createWrapper(ctx context.Context) (*sqlite3_wrap.Wrapper, error) {
	mem := &sqlite3_wrap.Memory{Max: 4096} // 256MB
	if strconv.IntSize < 64 {
		mem.Max = 512 // 32MB
	}
	if max, ok := ctx.Value(configKey{}).(int64); ok {
		mem.Max = max
	}
	mem.Grow(5, mem.Max) // 320KB
	env := &env{&sqlite3_wrap.Wrapper{Memory: mem}}
	env.Module = sqlite3_wasm.New(env)
	env.X_initialize()
	return env.Wrapper, nil
}

func (e *env) Xmemory() sqlite3_wasm.Memory { return e.Memory }

// VFS functions.

func (e *env) Xgo_randomness(pVfs, nByte, zByte int32) int32 {
	mem := e.Bytes(ptr_t(zByte), int64(nByte))
	n, _ := rand.Reader.Read(mem)
	return int32(n)
}

func (e *env) Xgo_sleep(pVfs, nMicro int32) int32 {
	time.Sleep(time.Duration(nMicro) * time.Microsecond)
	return _OK
}

func (e *env) Xgo_current_time_64(pVfs, nMicro int32) int32 {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	e.Write64(ptr_t(nMicro), uint64(msec))
	return int32(_OK)
}

func (e *env) Xgo_localtime(pTm int32, t int64) int32 {
	const size = 32 / 8
	mem := e.Memory
	tm := time.Unix(t, 0)
	// https://pubs.opengroup.org/onlinepubs/7908799/xsh/time.h.html
	mem.Write32(ptr_t(pTm+0*size), uint32(tm.Second()))
	mem.Write32(ptr_t(pTm+1*size), uint32(tm.Minute()))
	mem.Write32(ptr_t(pTm+2*size), uint32(tm.Hour()))
	mem.Write32(ptr_t(pTm+3*size), uint32(tm.Day()))
	mem.Write32(ptr_t(pTm+4*size), uint32(tm.Month()-time.January))
	mem.Write32(ptr_t(pTm+5*size), uint32(tm.Year()-1900))
	mem.Write32(ptr_t(pTm+6*size), uint32(tm.Weekday()-time.Sunday))
	mem.Write32(ptr_t(pTm+7*size), uint32(tm.YearDay()-1))
	mem.WriteBool(ptr_t(pTm+8*size), tm.IsDST())
	return _OK
}

func (e *env) Xgo_vfs_find(zVfsName int32) int32 {
	if vfs.Find(e.ReadString(ptr_t(zVfsName), _MAX_NAME)) != nil {
		return 1
	}
	return 0
}

//go:linkname vfsFullPathname github.com/ncruces/go-sqlite3/vfs.vfsFullPathname
func vfsFullPathname(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3 int32) int32

func (e *env) Xgo_full_pathname(v0, v1, v2, v3 int32) int32 {
	return vfsFullPathname(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsDelete github.com/ncruces/go-sqlite3/vfs.vfsDelete
func vfsDelete(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32) int32

func (e *env) Xgo_delete(v0, v1, v2 int32) int32 {
	return vfsDelete(e.Wrapper, v0, v1, v2)
}

//go:linkname vfsAccess github.com/ncruces/go-sqlite3/vfs.vfsAccess
func vfsAccess(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3 int32) int32

func (e *env) Xgo_access(v0, v1, v2, v3 int32) int32 {
	return vfsAccess(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsOpen github.com/ncruces/go-sqlite3/vfs.vfsOpen
func vfsOpen(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3, v4, v5 int32) int32

func (e *env) Xgo_open(v0, v1, v2, v3, v4, v5 int32) int32 {
	return vfsOpen(e.Wrapper, v0, v1, v2, v3, v4, v5)
}

//go:linkname vfsClose github.com/ncruces/go-sqlite3/vfs.vfsClose
func vfsClose(_ *sqlite3_wrap.Wrapper, v0 int32) int32

func (e *env) Xgo_close(v0 int32) int32 {
	return vfsClose(e.Wrapper, v0)
}

//go:linkname vfsRead github.com/ncruces/go-sqlite3/vfs.vfsRead
func vfsRead(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32, v3 int64) int32

func (e *env) Xgo_read(v0, v1, v2 int32, v3 int64) int32 {
	return vfsRead(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsWrite github.com/ncruces/go-sqlite3/vfs.vfsWrite
func vfsWrite(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32, v3 int64) int32

func (e *env) Xgo_write(v0, v1, v2 int32, v3 int64) int32 {
	return vfsWrite(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsTruncate github.com/ncruces/go-sqlite3/vfs.vfsTruncate
func vfsTruncate(_ *sqlite3_wrap.Wrapper, v0 int32, v1 int64) int32

func (e *env) Xgo_truncate(v0 int32, v1 int64) int32 {
	return vfsTruncate(e.Wrapper, v0, v1)
}

//go:linkname vfsSync github.com/ncruces/go-sqlite3/vfs.vfsSync
func vfsSync(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e *env) Xgo_sync(v0, v1 int32) int32 {
	return vfsSync(e.Wrapper, v0, v1)
}

//go:linkname vfsFileSize github.com/ncruces/go-sqlite3/vfs.vfsFileSize
func vfsFileSize(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e *env) Xgo_file_size(v0, v1 int32) int32 {
	return vfsFileSize(e.Wrapper, v0, v1)
}

//go:linkname vfsLock github.com/ncruces/go-sqlite3/vfs.vfsLock
func vfsLock(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e *env) Xgo_lock(v0 int32, v1 int32) int32 {
	return vfsLock(e.Wrapper, v0, v1)
}

//go:linkname vfsUnlock github.com/ncruces/go-sqlite3/vfs.vfsUnlock
func vfsUnlock(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e *env) Xgo_unlock(v0, v1 int32) int32 {
	return vfsUnlock(e.Wrapper, v0, v1)
}

//go:linkname vfsCheckReservedLock github.com/ncruces/go-sqlite3/vfs.vfsCheckReservedLock
func vfsCheckReservedLock(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e *env) Xgo_check_reserved_lock(v0, v1 int32) int32 {
	return vfsCheckReservedLock(e.Wrapper, v0, v1)
}

//go:linkname vfsFileControl github.com/ncruces/go-sqlite3/vfs.vfsFileControl
func vfsFileControl(_ *sqlite3_wrap.Wrapper, v0, v1, v2 int32) int32

func (e *env) Xgo_file_control(v0, v1, v2 int32) int32 {
	return vfsFileControl(e.Wrapper, v0, v1, v2)
}

//go:linkname vfsSectorSize github.com/ncruces/go-sqlite3/vfs.vfsSectorSize
func vfsSectorSize(_ *sqlite3_wrap.Wrapper, v0 int32) int32

func (e *env) Xgo_sector_size(v0 int32) int32 {
	return vfsSectorSize(e.Wrapper, v0)
}

//go:linkname vfsDeviceCharacteristics github.com/ncruces/go-sqlite3/vfs.vfsDeviceCharacteristics
func vfsDeviceCharacteristics(_ *sqlite3_wrap.Wrapper, v0 int32) int32

func (e *env) Xgo_device_characteristics(v0 int32) int32 {
	return vfsDeviceCharacteristics(e.Wrapper, v0)
}

//go:linkname vfsShmBarrier github.com/ncruces/go-sqlite3/vfs.vfsShmBarrier
func vfsShmBarrier(_ *sqlite3_wrap.Wrapper, v0 int32)

func (e *env) Xgo_shm_barrier(v0 int32) {
	vfsShmBarrier(e.Wrapper, v0)
}

//go:linkname vfsShmMap github.com/ncruces/go-sqlite3/vfs.vfsShmMap
func vfsShmMap(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3, v4 int32) int32

func (e *env) Xgo_shm_map(v0, v1, v2, v3, v4 int32) int32 {
	return vfsShmMap(e.Wrapper, v0, v1, v2, v3, v4)
}

//go:linkname vfsShmLock github.com/ncruces/go-sqlite3/vfs.vfsShmLock
func vfsShmLock(_ *sqlite3_wrap.Wrapper, v0, v1, v2, v3 int32) int32

func (e *env) Xgo_shm_lock(v0, v1, v2, v3 int32) int32 {
	return vfsShmLock(e.Wrapper, v0, v1, v2, v3)
}

//go:linkname vfsShmUnmap github.com/ncruces/go-sqlite3/vfs.vfsShmUnmap
func vfsShmUnmap(_ *sqlite3_wrap.Wrapper, v0, v1 int32) int32

func (e *env) Xgo_shm_unmap(v0, v1 int32) int32 {
	return vfsShmUnmap(e.Wrapper, v0, v1)
}
